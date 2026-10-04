package tool_test

import (
	"context"

	"github.com/maccavelli/gobble-cli/tool"
)

func ExampleNew() {
	type in struct {
		Path string
	}
	_ = tool.New[in, string]("read", "read a file", func(context.Context, in, tool.Env) (string, error) {
		return "", nil
	}, tool.WithKind(tool.KindRead))
}
