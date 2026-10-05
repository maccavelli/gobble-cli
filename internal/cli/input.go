package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// Prompt is the first turn's input: text, and any images given as @path.
type Prompt struct {
	Text   string
	Images []Image
}

// Image is an image @path, sent as an ACP image content block.
type Image struct {
	Name string // absolute path
	MIME string
	Data []byte
}

// imageTypes are the image types a prompt may carry.
var imageTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true}

// composeInput assembles the first prompt (0008-MADR D4) from three parts,
// joined by a blank line and each skipped when empty:
//  1. stdin, when it is not a terminal, read whole and trimmed;
//  2. each @path word: a text file wrapped as <file name="ABS">…</file>,
//     an image as an image part;
//  3. the other words, joined by single spaces.
//
// A missing @path is a usage error. A failed stdin read is an error.
func composeInput(ctx context.Context, stdin io.Reader, stdinTTY bool, words []string,
	readFile func(string) ([]byte, error)) (Prompt, error) {
	var parts []string
	var p Prompt
	if !stdinTTY && stdin != nil {
		b, err := readAll(ctx, stdin)
		if err != nil {
			return Prompt{}, failf("read stdin: %v", err)
		}
		if s := strings.TrimSpace(string(b)); s != "" {
			parts = append(parts, s)
		}
	}
	var plain []string
	for _, w := range words {
		if !isAtPath(w) {
			plain = append(plain, w)
			continue
		}
		abs, err := filepath.Abs(w[1:])
		if err != nil {
			return Prompt{}, usageErrorf("%s: %v", w, err)
		}
		data, err := readFile(abs)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return Prompt{}, usageErrorf("%s: no such file", w)
		case err != nil:
			return Prompt{}, failf("%s: %v", w, err)
		}
		if typ := imageType(abs, data); typ != "" {
			p.Images = append(p.Images, Image{Name: abs, MIME: typ, Data: data})
			continue
		}
		parts = append(parts, fmt.Sprintf("<file name=%q>\n%s\n</file>", abs, strings.TrimSuffix(string(data), "\n")))
	}
	if len(plain) > 0 {
		parts = append(parts, strings.Join(plain, " "))
	}
	p.Text = strings.Join(parts, "\n\n")
	return p, nil
}

// imageType is the MIME type of an image file, by content and then by
// extension, or "" for any other file.
func imageType(name string, data []byte) string {
	if t := http.DetectContentType(data); imageTypes[t] {
		return t
	}
	if t, _, _ := strings.Cut(mime.TypeByExtension(strings.ToLower(filepath.Ext(name))), ";"); imageTypes[t] {
		return t
	}
	return ""
}

// readAll reads r to the end unless ctx is done first.
func readAll(ctx context.Context, r io.Reader) ([]byte, error) {
	type result struct {
		b   []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		b, err := io.ReadAll(r)
		ch <- result{b, err}
	}()
	select {
	case res := <-ch:
		return res.b, res.err
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	}
}
