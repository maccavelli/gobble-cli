package render

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// diffContext is the number of unchanged lines around a change.
const diffContext = 3

// UnifiedDiff is the unified diff of old against new, as ACP diff content
// carries a change (0002-PLAN Phase 3 step 6): `--- a/<path>` and
// `+++ b/<path>` headers (`--- /dev/null` for a new file, old nil), then
// @@ hunks with three lines of context. An absolute or ~ path is named as
// it is, without the a/ and b/ of a relative one. It is a Myers line diff.
// Equal texts give "". The result is plain text; Diff colours it.
func UnifiedDiff(path string, old *string, newText string) string {
	var a []string
	from, to := diffLabel("a", path), diffLabel("b", path)
	if old == nil {
		from = "/dev/null"
	} else {
		a = splitLines(*old)
	}
	b := splitLines(newText)
	ops := myers(a, b)
	hunks := group(ops)
	if len(hunks) == 0 {
		return ""
	}
	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", from, to)
	for _, h := range hunks {
		writeHunk(&out, h, ops)
	}
	return out.String()
}

func diffLabel(side, path string) string {
	if strings.HasPrefix(path, "~") || strings.HasPrefix(path, "/") || filepath.IsAbs(path) {
		return path
	}
	return side + "/" + path
}

// splitLines splits s into lines, each with its "\n"; a last line without
// one is kept as it is.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.SplitAfter(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

type opKind byte

const (
	opEqual opKind = ' '
	opDel   opKind = '-'
	opIns   opKind = '+'
)

// op is one line of the edit script, with its line index in a (for equal
// and deleted lines) and in b (for equal and inserted lines).
type op struct {
	kind opKind
	text string
	ai   int
	bi   int
}

// myers is the shortest edit script from a to b (Myers, "An O(ND)
// Difference Algorithm and Its Variations", 1986), found forwards and
// rebuilt from the saved frontier of each step.
func myers(a, b []string) []op {
	n, m := len(a), len(b)
	maxD := n + m
	offset := maxD + 1
	v := make([]int, 2*maxD+3)
	var trace [][]int
	for d := 0; d <= maxD; d++ {
		trace = append(trace, append([]int(nil), v...))
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || k != d && v[offset+k-1] < v[offset+k+1] {
				x = v[offset+k+1] // down: an insertion
			} else {
				x = v[offset+k-1] + 1 // right: a deletion
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x, y = x+1, y+1
			}
			v[offset+k] = x
			if x >= n && y >= m {
				return backtrack(a, b, trace, offset, d)
			}
		}
	}
	return nil
}

func backtrack(a, b []string, trace [][]int, offset, d int) []op {
	x, y := len(a), len(b)
	var rev []op
	for ; d > 0; d-- {
		v := trace[d]
		k := x - y
		var prevK int
		if k == -d || k != d && v[offset+k-1] < v[offset+k+1] {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := v[offset+prevK]
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			x, y = x-1, y-1
			rev = append(rev, op{kind: opEqual, text: a[x], ai: x, bi: y})
		}
		if x == prevX {
			y--
			rev = append(rev, op{kind: opIns, text: b[y], ai: x, bi: y})
		} else {
			x--
			rev = append(rev, op{kind: opDel, text: a[x], ai: x, bi: y})
		}
	}
	for x > 0 && y > 0 {
		x, y = x-1, y-1
		rev = append(rev, op{kind: opEqual, text: a[x], ai: x, bi: y})
	}
	ops := make([]op, len(rev))
	for i, o := range rev {
		ops[len(rev)-1-i] = o
	}
	return ops
}

// hunk is a range of ops: changes and up to diffContext equal lines around
// them, two changes closer than twice that sharing one hunk.
type hunk struct{ start, end int }

func group(ops []op) []hunk {
	var hunks []hunk
	for i := 0; i < len(ops); i++ {
		if ops[i].kind == opEqual {
			continue
		}
		start := max(0, i-diffContext)
		end := i
		for j := i; j < len(ops); j++ {
			if ops[j].kind != opEqual {
				end = j
				continue
			}
			if j-end > 2*diffContext {
				break
			}
		}
		end = min(len(ops), end+1+diffContext)
		if n := len(hunks); n > 0 && start <= hunks[n-1].end {
			hunks[n-1].end = end
		} else {
			hunks = append(hunks, hunk{start: start, end: end})
		}
		i = end - 1
	}
	return hunks
}

func writeHunk(out *strings.Builder, h hunk, ops []op) {
	aStart, bStart, aLen, bLen := -1, -1, 0, 0
	for _, o := range ops[h.start:h.end] {
		if o.kind != opIns {
			if aStart < 0 {
				aStart = o.ai
			}
			aLen++
		}
		if o.kind != opDel {
			if bStart < 0 {
				bStart = o.bi
			}
			bLen++
		}
	}
	first := ops[h.start]
	if aStart < 0 {
		aStart = first.ai
	}
	if bStart < 0 {
		bStart = first.bi
	}
	fmt.Fprintf(out, "@@ -%s +%s @@\n", hunkRange(aStart, aLen), hunkRange(bStart, bLen))
	for _, o := range ops[h.start:h.end] {
		out.WriteByte(byte(o.kind))
		out.WriteString(o.text)
		if !strings.HasSuffix(o.text, "\n") {
			out.WriteString("\n\\ No newline at end of file\n")
		}
	}
}

// hunkRange is a hunk header's start,length, 1-based; an empty range names
// the line before it, as diff -u does.
func hunkRange(start, n int) string {
	if n == 0 {
		return fmt.Sprintf("%d,0", start)
	}
	if n == 1 {
		return strconv.Itoa(start + 1)
	}
	return fmt.Sprintf("%d,%d", start+1, n)
}
