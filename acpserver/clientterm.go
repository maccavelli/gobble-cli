package acpserver

import (
	"context"
	"strings"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/tool"
)

// clientTerminal is a session's client terminals, through terminal/*
// (0005-PLAN F1d-3). created is told each new terminal's id, so the
// running tool call can carry it and the client shows the terminal live.
type clientTerminal struct {
	conn    *acp.AgentSideConnection
	session acp.SessionId
	created func(id string)
}

var _ tool.Terminal = clientTerminal{}

func (t clientTerminal) Create(ctx context.Context, r tool.TerminalRequest) (string, error) {
	req := acp.CreateTerminalRequest{SessionId: t.session, Command: r.Command, Args: r.Args}
	if r.Cwd != "" {
		req.Cwd = &r.Cwd
	}
	for _, kv := range r.Env {
		name, value, _ := strings.Cut(kv, "=")
		req.Env = append(req.Env, acp.EnvVariable{Name: name, Value: value})
	}
	if r.OutputByteLimit > 0 {
		req.OutputByteLimit = &r.OutputByteLimit
	}
	resp, err := t.conn.CreateTerminal(ctx, req)
	if err != nil {
		return "", err
	}
	if t.created != nil {
		t.created(resp.TerminalId)
	}
	return resp.TerminalId, nil
}

func (t clientTerminal) Output(ctx context.Context, id string) (tool.TerminalOutput, error) {
	r, err := t.conn.TerminalOutput(ctx, acp.TerminalOutputRequest{SessionId: t.session, TerminalId: id})
	if err != nil {
		return tool.TerminalOutput{}, err
	}
	out := tool.TerminalOutput{Output: r.Output, Truncated: r.Truncated}
	if r.ExitStatus != nil {
		out.Exit = &tool.TerminalExit{Code: r.ExitStatus.ExitCode, Signal: deref(r.ExitStatus.Signal)}
	}
	return out, nil
}

func (t clientTerminal) WaitForExit(ctx context.Context, id string) (tool.TerminalExit, error) {
	r, err := t.conn.WaitForTerminalExit(ctx, acp.WaitForTerminalExitRequest{SessionId: t.session, TerminalId: id})
	if err != nil {
		return tool.TerminalExit{}, err
	}
	return tool.TerminalExit{Code: r.ExitCode, Signal: deref(r.Signal)}, nil
}

func (t clientTerminal) Kill(ctx context.Context, id string) error {
	_, err := t.conn.KillTerminal(ctx, acp.KillTerminalRequest{SessionId: t.session, TerminalId: id})
	return err
}

func (t clientTerminal) Release(ctx context.Context, id string) error {
	_, err := t.conn.ReleaseTerminal(ctx, acp.ReleaseTerminalRequest{SessionId: t.session, TerminalId: id})
	return err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
