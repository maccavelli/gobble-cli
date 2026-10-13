package editor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Compose writes initial to a new temp file in dir, opens it in the user's
// editor and returns what the file holds afterwards, without the trailing
// newline. The editor is GOBBLE_EDITOR, configured (the externalEditor
// setting), VISUAL or EDITOR, else notepad on Windows and vi elsewhere
// (0018-MADR D17). It runs without a shell, through run, which
// attaches the terminal (exec.Cmd.Run with the process's streams, in
// production). The temp file has an unpredictable name and is removed.
func Compose(ctx context.Context, env func(string) string, configured, dir, initial string, run func(*exec.Cmd) error) (text string, err error) {
	argv, err := editorCommand(env, runtime.GOOS, configured)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, "gobble-prompt-*.md")
	if err != nil {
		return "", fmt.Errorf("compose: %w", err)
	}
	name := f.Name()
	defer func() { err = errors.Join(err, os.Remove(name)) }()
	if _, err := f.WriteString(initial); err != nil {
		return "", errors.Join(fmt.Errorf("compose: %w", err), f.Close())
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("compose: %w", err)
	}

	cmd := exec.CommandContext(ctx, argv[0], append(argv[1:], name)...) //nolint:gosec // the user's own editor, run without a shell
	if err := run(cmd); err != nil {
		return "", fmt.Errorf("editor %s: %w", argv[0], err)
	}
	b, err := os.ReadFile(name) //nolint:gosec // G304: the temp file created above
	if err != nil {
		return "", fmt.Errorf("compose: %w", err)
	}
	return strings.TrimRight(string(b), "\r\n"), nil
}

// Open opens file in the user's editor, chosen as Compose chooses it, and
// waits for it to exit (0018-PLAN P4 deviation 9). It runs without a
// shell, through run.
func Open(ctx context.Context, env func(string) string, configured, file string, run func(*exec.Cmd) error) error {
	argv, err := editorCommand(env, runtime.GOOS, configured)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, argv[0], append(argv[1:], file)...) //nolint:gosec // the user's own editor, run without a shell
	if err := run(cmd); err != nil {
		return fmt.Errorf("editor %s: %w", argv[0], err)
	}
	return nil
}

// editorCommand is the editor chain: GOBBLE_EDITOR, then configured, then
// VISUAL and EDITOR, then the platform's.
func editorCommand(env func(string) string, goos, configured string) ([]string, error) {
	for _, v := range []string{env("GOBBLE_EDITOR"), configured, env("VISUAL"), env("EDITOR")} {
		if v := strings.TrimSpace(v); v != "" {
			return splitCommand(v)
		}
	}
	if goos == "windows" {
		return []string{"notepad"}, nil
	}
	return []string{"vi"}, nil
}

// splitCommand splits an editor command the way goose does
// (crates/goose-cli/src/session/editor.rs, split_editor_command): on
// whitespace when there is no quote, so a Windows path keeps its
// backslashes; otherwise quote-aware, where single or double quotes group
// words. Backslashes are always literal.
func splitCommand(s string) ([]string, error) {
	if !strings.ContainsAny(s, `"'`) {
		return strings.Fields(s), nil
	}
	var (
		words  []string
		cur    strings.Builder
		inWord bool
		quote  rune
	)
	for _, r := range s {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			cur.WriteRune(r)
		case r == '"' || r == '\'':
			quote, inWord = r, true
		case r == ' ' || r == '\t':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("editor command %q: unmatched quote", s)
	}
	if inWord {
		words = append(words, cur.String())
	}
	if len(words) == 0 || words[0] == "" {
		return nil, fmt.Errorf("editor command %q is empty", s)
	}
	return words, nil
}
