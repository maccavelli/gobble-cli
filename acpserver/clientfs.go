package acpserver

import (
	"context"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/tool"
)

// clientFiles is a session's client files, through fs/read_text_file and
// fs/write_text_file (0005-PLAN F1d-3). The tools have resolved and
// confined each path before they ask.
type clientFiles struct {
	conn    *acp.AgentSideConnection
	session acp.SessionId
}

var _ tool.TextFiles = clientFiles{}

func (f clientFiles) ReadTextFile(ctx context.Context, path string) (string, error) {
	r, err := f.conn.ReadTextFile(ctx, acp.ReadTextFileRequest{SessionId: f.session, Path: path})
	if err != nil {
		return "", err
	}
	return r.Content, nil
}

func (f clientFiles) WriteTextFile(ctx context.Context, path, content string) error {
	_, err := f.conn.WriteTextFile(ctx, acp.WriteTextFileRequest{SessionId: f.session, Path: path, Content: content})
	return err
}
