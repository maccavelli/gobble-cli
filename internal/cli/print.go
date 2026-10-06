package cli

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// closeWait bounds session/close and the agent's shutdown after a turn.
const closeWait = 2 * time.Second

// printResult is the json format's object and stream-json's last line
// (0008-MADR D9, 0005-MADR "Print mode"). Error is set only when
// session/prompt failed; stopReason is then "error" (owner's decision of
// 2026-10-05).
type printResult struct {
	Type       string           `json:"type,omitzero"`
	Text       string           `json:"text"`
	StopReason string           `json:"stopReason"`
	Usage      *acpclient.Usage `json:"usage"`
	Cost       *acpclient.Cost  `json:"cost"`
	SessionID  string           `json:"sessionId"`
	Error      string           `json:"error,omitzero"`
}

// headerType and resultType are stream-json's own line types.
const (
	headerType = "session"
	resultType = "result"
)

// sessionHeader is stream-json's first line.
type sessionHeader struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Cwd       string `json:"cwd"`
	Timestamp string `json:"timestamp"`
}

// printSink collects a print-mode turn. In stream-json it writes each
// update's params as one line, holding back any that arrive before the
// session header is written.
type printSink struct {
	out    *Output
	stream bool
	now    func() time.Time

	mu      sync.Mutex
	text    strings.Builder
	cost    *acpclient.Cost
	usage   *acpclient.ContextUsage
	clock   turnClock
	started bool
	held    [][]byte
	err     error
}

func (s *printSink) on(u acpclient.Update) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch u.Kind {
	case acpclient.KindAgentText:
		s.clock.token(s.now())
		s.text.WriteString(u.Text)
	case acpclient.KindThought:
		s.clock.token(s.now())
	case acpclient.KindUsage:
		c := u.Context
		s.usage = &c
		if c.Cost != nil {
			s.cost = c.Cost
		}
	}
	if !s.stream {
		return
	}
	line := append(append([]byte(nil), u.Params...), '\n')
	if !s.started {
		s.held = append(s.held, line)
		return
	}
	s.write(line)
}

func (s *printSink) write(b []byte) {
	if s.err == nil {
		s.err = s.out.Result(b)
	}
}

// begin starts the turn's clock and writes stream-json's header and the
// updates held back for it.
func (s *printSink) begin(id, cwd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started = true
	s.clock.start = s.now()
	if !s.stream {
		return
	}
	s.writeJSON(sessionHeader{Type: headerType, ID: id, Cwd: cwd, Timestamp: s.now().UTC().Format(time.RFC3339)})
	for _, b := range s.held {
		s.write(b)
	}
	s.held = nil
}

func (s *printSink) writeJSON(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		if s.err == nil {
			s.err = fmt.Errorf("encode output: %w", err)
		}
		return
	}
	s.write(append(b, '\n'))
}

// runPrint runs one turn and prints it in the chosen format (0008-MADR D9),
// mapping the outcome to D10's codes.
func runPrint(e *runEnv, c *ChatCmd, p Prompt, cwd string) (err error) {
	ctx := e.ctx
	conn, err := e.startAgent(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			e.out.Warnf("%v", cerr)
		}
	}()
	sink := &printSink{out: e.out, stream: c.OutputFormat == "stream-json", now: e.now}
	sess, err := conn.NewSession(ctx, cwd, sink.on)
	if err != nil {
		return failf("%v", err)
	}
	defer func() {
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), closeWait)
		defer cancel()
		if cerr := sess.Close(cctx); cerr != nil && ctx.Err() == nil {
			e.out.Warnf("%v", cerr)
		}
	}()
	sink.begin(sess.ID(), cwd)
	// A cancelled ctx (a signal) makes the SDK send session/cancel itself.
	res, perr := sess.Prompt(ctx, clientPrompt(p))
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	if errors.Is(perr, acpclient.ErrImagesUnsupported) {
		return failf("the agent does not accept images (%s)", imageNames(p))
	}
	return finishPrint(e, c, sink, sess.ID(), res, perr)
}

// finishPrint writes the result and decides the exit: a failed request and
// an unrequested cancel exit 1 (D10).
func finishPrint(e *runEnv, c *ChatCmd, sink *printSink, id string, res acpclient.Result, perr error) error {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	sink.clock.end = e.now()
	text := term.Sanitize(sink.text.String())
	r := printResult{Text: text, StopReason: res.StopReason, Usage: res.Usage, Cost: sink.cost, SessionID: id}
	if perr != nil {
		r.StopReason, r.Error = "error", perr.Error()
	}
	switch c.OutputFormat {
	case "json":
		sink.writeJSON(r)
	case "stream-json":
		r.Type = resultType
		sink.writeJSON(r)
	default:
		if text != "" && perr == nil {
			sink.write([]byte(strings.TrimRight(text, "\n") + "\n"))
		}
	}
	if sink.err != nil {
		return sink.err
	}
	if c.Stats {
		line := statusLine(sink.clock, sink.usage, res, true, e.out.caps.Err, glyphsFor(e.out.caps))
		if _, err := io.WriteString(e.out.err, dimmed(line, e.out.caps.Err.Color)+"\n"); err != nil {
			return fmt.Errorf("write stderr: %w", err)
		}
	}
	switch {
	case errors.Is(perr, acpclient.ErrAuthRequired):
		return &exitError{code: ExitAuth, msg: noCredentialMessage()}
	case perr != nil:
		return failf("%v", perr)
	case res.StopReason == "cancelled":
		return failf("the agent cancelled the turn")
	}
	return nil
}

// clientPrompt is the composed input as ACP content.
func clientPrompt(p Prompt) acpclient.Prompt {
	cp := acpclient.Prompt{Text: p.Text}
	for _, img := range p.Images {
		cp.Images = append(cp.Images, acpclient.Image{MIME: img.MIME, Data: img.Data})
	}
	return cp
}

func imageNames(p Prompt) string {
	names := make([]string, 0, len(p.Images))
	for _, img := range p.Images {
		names = append(names, img.Name)
	}
	return strings.Join(names, ", ")
}
