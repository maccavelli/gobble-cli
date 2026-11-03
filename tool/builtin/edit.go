package builtin

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/tool"
)

type editIn struct {
	Path  string `json:"path" jsonschema:"the file to edit, relative to the working directory or absolute"`
	Edits []edit `json:"edits" jsonschema:"one or more replacements, each matched against the file as it was before this call; they must not overlap, so merge changes to the same lines into one edit"`
}

// beforeWrite runs between an edit's computation and its compare-and-swap.
// Only tests set it, to change the file underneath an edit.
var beforeWrite func(abs string)

// errChanged is the compare-and-swap's failure.
var errChanged = errors.New("the file changed while the edit was being made; read it again and resend the edit")

// Edit is the edit tool: several replacements in one file, each matched
// against the original, with its BOM and line endings kept and forgiving
// matching (editmatch.go). Under the file's lock, the bytes it read are
// compared with the file again just before the write, so a change by
// another process fails the edit instead of being overwritten.
func Edit() tool.Tool {
	return tool.New("edit", description("edit", Options{}.withDefaults()),
		func(ctx context.Context, in editIn, env tool.Env) (tool.Result, error) {
			name := shown(env, in.Path)
			if len(in.Edits) == 0 {
				return tool.Result{}, errors.New("edit: edits needs at least one replacement")
			}
			abs := resolve(env, in.Path)
			unlock := fsx.Lock(abs)
			defer unlock()
			io := textIOFor(ctx, env, workspace(env), abs)
			b, err := io.read(abs)
			if err != nil {
				return tool.Result{}, fmt.Errorf("edit %s: %w", name, unwrapPath(err))
			}
			raw := string(b)
			bom, content := "", raw
			if rest, ok := strings.CutPrefix(raw, "\ufeff"); ok {
				bom, content = "\ufeff", rest
			}
			updated, les, err := applyEdits(lfView(content), in.Edits, name)
			if err != nil {
				return tool.Result{}, err
			}
			final := bom + updated
			if strings.Contains(content, "\r") {
				// Only the lines the edits wrote take the file's ending; every
				// other line keeps its own (0037-MADR D8).
				final = bom + spliceLines(linesWithEndings(content), les, lineEnding(content))
			}
			if beforeWrite != nil {
				beforeWrite(abs)
			}
			if now, err := io.read(abs); err != nil || !bytes.Equal(now, b) {
				return tool.Result{}, fmt.Errorf("edit %s: %w", name, errChanged)
			}
			if err := io.write(abs, []byte(final)); err != nil {
				return tool.Result{}, fmt.Errorf("edit %s: %w", name, unwrapPath(err))
			}
			msg := fmt.Sprintf("edited %s: %s", name, replacements(len(in.Edits)))
			r := tool.TextResult(msg)
			r.Summary = summary(msg)
			r.Diffs = []tool.Diff{{Path: abs, OldText: &raw, NewText: final}}
			return r, nil
		},
		tool.WithKind(tool.KindEdit),
		tool.WithAnnotations(tool.Annotations{DestructiveHint: true}),
		tool.WithOutside(outsidePath),
		tool.WithSubject(pathSubject("path")),
		tool.WithPrepare(prepareEdit),
		tool.WithSchema(func(s *jsonschema.Schema) {
			edits := property(s, "edits")
			edits.MinItems = new(1)
			nonEmpty(edits.Items, "oldText")
		}),
		tool.WithDescribe(func(c tool.Call, env tool.Env) string { return title("edit " + shown(env, titlePath(c))) }),
		tool.WithLocations(argLocations("path")),
	)
}

// replacements is "1 replacement" or "N replacements".
func replacements(n int) string {
	if n == 1 {
		return "1 replacement"
	}
	return fmt.Sprintf("%d replacements", n)
}

// prepareEdit repairs three shapes models send, as Pi's edit tool does:
// edits as a JSON string is parsed, a single edit object becomes a list of
// one, and top-level oldText and newText are appended to edits.
func prepareEdit(args jsontext.Value) (jsontext.Value, error) {
	m, ok := object(args)
	if !ok {
		return args, nil // not an object: validation says so
	}
	if raw, ok := m["edits"]; ok {
		m["edits"] = editList(raw)
	}
	var oldText, newText string
	if jsonx.Unmarshal(m["oldText"], &oldText) == nil && jsonx.Unmarshal(m["newText"], &newText) == nil {
		var edits []jsontext.Value
		if err := jsonx.Unmarshal(m["edits"], &edits); err != nil {
			edits = nil
		}
		single, err := jsonx.Marshal(edit{OldText: oldText, NewText: newText})
		if err != nil {
			return nil, err
		}
		edits = append(edits, single)
		list, err := jsonx.Marshal(edits)
		if err != nil {
			return nil, err
		}
		m["edits"] = list
		delete(m, "oldText")
		delete(m, "newText")
	}
	return jsonx.Marshal(m)
}

// object is args as an object's members, when args is one.
func object(args jsontext.Value) (map[string]jsontext.Value, bool) {
	var m map[string]jsontext.Value
	if err := jsonx.Unmarshal(args, &m); err != nil || m == nil {
		return nil, false
	}
	return m, true
}

// editList is edits as a list: a JSON string holding a list or one edit is
// parsed, and one edit object becomes a list of one. Anything else is left
// for validation to refuse.
func editList(raw jsontext.Value) jsontext.Value {
	candidate := raw
	var s string
	fromString := jsonx.Unmarshal(raw, &s) == nil
	if fromString {
		candidate = jsontext.Value(s)
	}
	if !candidate.IsValid() {
		return raw
	}
	switch candidate.Kind() {
	case '[':
		return candidate
	case '{':
		var one struct {
			OldText *string `json:"oldText"`
			NewText *string `json:"newText"`
		}
		if jsonx.Unmarshal(candidate, &one) == nil && one.OldText != nil && one.NewText != nil {
			if b, err := jsonx.Marshal([]edit{{OldText: *one.OldText, NewText: *one.NewText}}); err == nil {
				return b
			}
		}
	}
	return raw
}
