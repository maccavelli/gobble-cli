package acpserver

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/personality"
	"github.com/maccavelli/gobble-cli/session"
)

// methodPersonality is the notification of a session's personality, at its
// start and at each change (0022-MADR D6).
const methodPersonality = "_gobble/personality"

// personalityNotice is _gobble/personality's params.
type personalityNotice struct {
	SessionID acp.SessionId `json:"sessionId"`
	Name      string        `json:"name"`
	Level     int           `json:"level"`
}

// personalityData is a gobble.personality entry's data: the personality a
// switch chose, by its canonical name (0022-MADR D6).
type personalityData struct {
	Name string `json:"name"`
}

func personalityEntry(name string) session.Entry {
	return session.Entry{Type: session.TypeCustom, CustomType: session.CustomPersonality, Data: mustJSON(personalityData{Name: name})}
}

// personalityOf is a gobble.personality entry's name.
func personalityOf(e session.Entry) (string, bool) {
	if e.Type != session.TypeCustom || e.CustomType != session.CustomPersonality {
		return "", false
	}
	var d personalityData
	if err := json.Unmarshal(e.Data, &d); err != nil || d.Name == "" {
		return "", false
	}
	return d.Name, true
}

// The level-5 question's options (0022-MADR D10).
const (
	insultsSwitch = "switch"
	insultsStay   = "stay"
)

// InsultsCallID is the level-5 question's tool call id: the question is no
// tool call, and ACP's permission request needs one. gobble's print mode
// answers it no by this id (0022-MADR D10).
const InsultsCallID = "gobble-personality"

// voice is the session's voice, which a switch changes while a turn may
// read it.
func (a *Agent) voice(s *liveSession) *personality.Voice {
	a.mu.Lock()
	defer a.mu.Unlock()
	return s.voice
}

// newVoice is the named personality's voice: a built-in, one of boring's
// names, or one of Options.Personalities.
func (a *Agent) newVoice(name string) (*personality.Voice, error) {
	return a.personalities.New(personality.Options{Name: name})
}

// namedVoice is newVoice of a name as the user typed it: as it is, or in
// lower case.
func (a *Agent) namedVoice(name string) (*personality.Voice, error) {
	v, err := a.newVoice(name)
	if err != nil {
		if low := strings.ToLower(name); low != name {
			return a.newVoice(low)
		}
	}
	return v, err
}

// accepted reports whether the user has agreed to a level-5 personality:
// in the settings the session started with, or since, in this process
// (0022-MADR D10).
func (a *Agent) accepted(s *liveSession) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.insultsAccepted || s.insultsAccepted
}

// personalityNotice is the session's _gobble/personality.
func (a *Agent) personalityNotice(s *liveSession) personalityNotice {
	v := a.voice(s)
	return personalityNotice{SessionID: s.id, Name: v.Name(), Level: v.Level()}
}

// askInsults asks the client, once per user, before a level-5 personality
// speaks: a permission request, in plain words, since a permission
// question is never in a voice (0022-MADR D2, D10). A yes is saved as
// bombasticAccepted. ok is false when the user declined, or nobody could
// answer; err is a cancelled turn's, or the save's.
func (a *Agent) askInsults(ctx context.Context, conn *acp.AgentSideConnection, s *liveSession, v *personality.Voice) (bool, error) {
	if v.Level() < personality.InsultLevel || a.accepted(s) {
		return true, nil
	}
	if conn == nil {
		return false, nil
	}
	resp, err := conn.RequestPermission(ctx, acp.RequestPermissionRequest{
		SessionId: s.id,
		ToolCall:  acp.ToolCallUpdate{ToolCallId: InsultsCallID, Title: new(insultsQuestion(v.Name()))},
		Options: []acp.PermissionOption{
			{OptionId: insultsSwitch, Name: "Switch to " + v.Name(), Kind: acp.PermissionOptionKindAllowAlways},
			{OptionId: insultsStay, Name: "No", Kind: acp.PermissionOptionKindRejectOnce},
		},
	})
	if ctx.Err() != nil {
		return false, context.Cause(ctx)
	}
	if err != nil {
		a.logger.WarnContext(ctx, "personality: the question was not answered", slog.String("error", err.Error()))
		return false, nil
	}
	if sel := resp.Outcome.Selected; sel == nil || sel.OptionId != insultsSwitch {
		return false, nil
	}
	if a.save != nil {
		if err := a.save("bombasticAccepted", "true"); err != nil {
			return false, acp.NewInternalError(map[string]any{keyReason: err.Error()})
		}
	}
	a.mu.Lock()
	a.insultsAccepted = true
	a.mu.Unlock()
	return true, nil
}

// insultsQuestion is D10's question, without its "?", which the client's
// prompt adds.
func insultsQuestion(name string) string {
	return capitalized(name) + " insults you and your decisions. It does what you choose, and it keeps gobble's house rules. Switch to it"
}

func capitalized(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

// switchPersonality is /personality's, /humor's and
// _gobble/set_personality's one path: the level-5 question when it is
// owed, then the session's voice, written as a gobble.personality entry
// and reported as _gobble/personality. It returns the session's voice
// after it, and whether the switch was made.
func (a *Agent) switchPersonality(ctx context.Context, conn *acp.AgentSideConnection, s *liveSession, out *updates, v *personality.Voice) (*personality.Voice, bool, error) {
	ok, err := a.askInsults(ctx, conn, s, v)
	if err != nil || !ok {
		return a.voice(s), false, err
	}
	if err := s.append(ctx, a.now(), personalityEntry(v.Name())); err != nil {
		return nil, false, appendFailed(err)
	}
	a.mu.Lock()
	s.voice = v
	a.mu.Unlock()
	out.notify(methodPersonality, a.personalityNotice(s))
	return v, true, out.err
}

// settleVoice asks the level-5 question before a session that starts, or
// was resumed, at level 5 speaks, when the user has not agreed; a no, or
// no answer, switches it to dry. The reply says nothing of it: the client
// showed the question and its answer, and print mode's reply is the
// model's alone (0022-MADR D10).
func (a *Agent) settleVoice(ctx context.Context, conn *acp.AgentSideConnection, s *liveSession) error {
	v := a.voice(s)
	ok, err := a.askInsults(ctx, conn, s, v)
	if err != nil || ok {
		return err
	}
	dry, err := a.newVoice(personality.Default)
	if err != nil {
		return err
	}
	_, _, err = a.switchPersonality(ctx, conn, s, newUpdates(ctx, conn, s.id), dry)
	return err
}

// The commands settleVoice does not wait for: they choose the voice.
const (
	cmdPersonality = "personality"
	cmdHumor       = "humor"
)

// humorLevels are /humor's levels, the built-ins' (0022-MADR D1).
var humorLevels = map[string]string{"1": personality.Boring, "3": personality.Dry, "4": personality.Snarky, "5": personality.Bombastic}

// cmdPersonality is /personality: with no name, the session's personality
// and the list, with each one's level and summary; with one, a switch to
// it, the level-5 question first (0022-MADR D6, D10).
func (a *Agent) cmdPersonality(ctx context.Context, t *cmdTurn, arg string) error {
	if arg == "" {
		var b strings.Builder
		b.WriteString(t.text(personality.PersonalityCurrent, personality.Args{argName: t.voice.Name()}) + "\n")
		b.WriteString(t.text(personality.PersonalityList, nil) + "\n")
		for _, info := range a.personalities.Infos() {
			fmt.Fprintf(&b, "  %s (%d) — %s\n", info.Name, info.Level, info.Summary)
		}
		t.say(b.String())
		return nil
	}
	v, err := a.namedVoice(arg)
	if err != nil {
		t.say(t.text(personality.PersonalityUnknown, personality.Args{argName: strconv.Quote(arg),
			argNames: strings.Join(a.personalities.Names(), ", ")}))
		return nil //nolint:nilerr // the unknown name is the command's answer (above)
	}
	return a.choose(ctx, t, v)
}

// cmdHumor is the hidden /humor N: the built-in of level N.
func (a *Agent) cmdHumor(ctx context.Context, t *cmdTurn, arg string) error {
	name, ok := humorLevels[arg]
	switch {
	case arg == "":
		t.say(t.text(personality.PersonalityCurrent, personality.Args{argName: t.voice.Name()}))
		return nil
	case arg == "2":
		t.say(t.text(personality.HumorTwo, nil))
		return nil
	case !ok:
		t.say(t.text(personality.HumorUnknown, personality.Args{argLevel: strconv.Quote(arg)}))
		return nil
	}
	v, err := a.newVoice(name)
	if err != nil {
		return err
	}
	return a.choose(ctx, t, v)
}

// choose switches the command's session to v, and answers in the voice
// the session then has.
func (a *Agent) choose(ctx context.Context, t *cmdTurn, v *personality.Voice) error {
	before := t.voice
	after, switched, err := a.switchPersonality(ctx, t.conn, t.s, t.out, v)
	if err != nil {
		return err
	}
	t.voice = after
	if !switched {
		t.say(t.text(personality.PersonalityDeclined, personality.Args{argName: before.Name()}))
		return nil
	}
	t.say(t.text(personality.PersonalitySet, personality.Args{argName: after.Name()}))
	return nil
}

type setPersonalityParams struct {
	sessionParams `json:",inline"`
	Name          string `json:"name"`
}

// personalityResult is _gobble/set_personality's result: the session's
// personality after it, which a declined question leaves as it was.
type personalityResult struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// extSetPersonality is /personality NAME's path for a client: an unknown
// name is invalid params; a declined level-5 question is no error, and the
// result names the personality that stays (0022-MADR D6).
func (a *Agent) extSetPersonality(ctx context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[setPersonalityParams](a, params)
	if err != nil {
		return nil, err
	}
	v, err := a.namedVoice(p.Name)
	if err != nil || p.Name == "" {
		return nil, invalidParams(s.id, fmt.Sprintf("unknown personality %q; the personalities are %s", p.Name, strings.Join(a.personalities.Names(), ", ")))
	}
	conn := a.connection()
	after, _, err := a.switchPersonality(ctx, conn, s, newUpdates(ctx, conn, s.id), v)
	if err != nil {
		return nil, err
	}
	return personalityResult{Name: after.Name(), Level: after.Level()}, nil
}
