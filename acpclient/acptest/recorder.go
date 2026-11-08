package acptest

import (
	"encoding/json/jsontext"
	"path/filepath"
	"regexp"
	"strings"

	sdkacptest "github.com/maccavelli/acp-go-sdk2/acptest"

	"github.com/maccavelli/gobble-cli/jsonx"
)

// jsonString matches a JSON string literal in a recorded frame.
var jsonString = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)

// HostIndependent returns a copy of records with every JSON string in a frame
// that is a path under Abs("/") written in slash form, as the fixture wrote
// it: on Windows "C:\\work\\a.go" becomes "/work/a.go". Off Windows Abs("/")
// is "/" and the frames are unchanged. It is the one step gobble adds before
// acp-go-sdk2's acptest.Golden, which compares frames as they were
// recorded, so that a golden recorded on one host matches on another
// (0019-MADR D4; 0016-MADR D6).
func HostIndependent(records []sdkacptest.Record) []sdkacptest.Record {
	return slashRecords(records, Abs("/"))
}

// slashRecords is HostIndependent with root, the host's filesystem root.
func slashRecords(records []sdkacptest.Record, root string) []sdkacptest.Record {
	out := make([]sdkacptest.Record, len(records))
	for i, r := range records {
		out[i] = sdkacptest.Record{Direction: r.Direction, Frame: []byte(slashPaths(string(r.Frame), root))}
	}
	return out
}

// slashPaths is frame with each JSON string that is a path under root
// rewritten as "/" and the rest in slash form; with root "/" it is frame.
func slashPaths(frame, root string) string {
	if root == "/" {
		return frame
	}
	return jsonString.ReplaceAllStringFunc(frame, func(lit string) string {
		var s string
		if jsonx.Unmarshal([]byte(lit), &s) != nil {
			return lit
		}
		rest, ok := strings.CutPrefix(s, root)
		if !ok {
			return lit
		}
		b, err := jsontext.AppendQuote(nil, "/"+strings.ReplaceAll(filepath.ToSlash(rest), `\`, "/"))
		if err != nil {
			return lit
		}
		return string(b)
	})
}
