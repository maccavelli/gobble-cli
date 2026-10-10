package session

import (
	"encoding/json/jsontext"
	"fmt"
	"maps"
	"slices"
	"sync"
)

// Tree is a session's entries with a leaf that can move, as Pi's
// SessionManager keeps them (session-manager.ts): the leaf is the entry
// the next one is a child of, and Path walks from it to the root. NewTree
// puts the leaf on the last entry in file order, as a load does; Branch,
// ResetLeaf and a branch summary move it. An entry joins the tree only
// through Add, once it is written, so a failed write leaves the tree as it
// was. A Tree is safe for concurrent use.
type Tree struct {
	mu      sync.Mutex
	entries []Entry
	byID    map[string]int // the last entry with each id, as Pi's index keeps it
	leaf    int            // the leaf's index, or -1 when there is none
}

// Label is a target's label, as the label entries resolve it.
type Label struct {
	TargetID  string
	Label     string
	Timestamp string // that of the label entry that set it
}

// NewTree is the tree of entries, its leaf on the last of them.
func NewTree(entries []Entry) *Tree {
	t := &Tree{entries: slices.Clone(entries), byID: make(map[string]int, len(entries)), leaf: len(entries) - 1}
	for i, e := range t.entries {
		t.byID[e.ID] = i
	}
	return t
}

// Leaf is the leaf's id, or "" when there is none.
func (t *Tree) Leaf() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.leafID()
}

func (t *Tree) leafID() string {
	if t.leaf < 0 {
		return ""
	}
	return t.entries[t.leaf].ID
}

// Entries are every entry, in file order.
func (t *Tree) Entries() []Entry {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.entries)
}

// Path is the active path, root first: from the leaf to its root, or nothing
// when there is no leaf.
func (t *Tree) Path() []Entry {
	t.mu.Lock()
	defer t.mu.Unlock()
	return pathFrom(t.entries, t.byID, t.leaf)
}

// Next is e stamped as the leaf's child, with a new id. It is not added.
func (t *Tree) Next(e Entry) Entry {
	t.mu.Lock()
	defer t.mu.Unlock()
	e.ID, e.ParentID = t.newID(nil), nil
	if t.leaf >= 0 {
		e.ParentID = new(t.leafID())
	}
	return e
}

// Add records a written entry and makes it the leaf.
func (t *Tree) Add(e Entry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.byID[e.ID] = len(t.entries)
	t.entries = append(t.entries, e)
	t.leaf = len(t.entries) - 1
}

// Branch moves the leaf to the entry id (Pi's branch), so the next entry
// starts a branch from it.
func (t *Tree) Branch(id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	i, ok := t.byID[id]
	if !ok {
		return fmt.Errorf("session: entry %s not found", id)
	}
	t.leaf = i
	return nil
}

// ResetLeaf removes the leaf (Pi's resetLeaf): the next entry is a root,
// and the path is empty.
func (t *Tree) ResetLeaf() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.leaf = -1
}

// BranchSummary is Pi's branchWithSummary: a stamped branch_summary whose
// parent is from, or none when from is "", the root, and whose fromId is
// the leaf it leaves, or "root" when there is none. It becomes the leaf
// through Add, once it is written. An unknown from is an error, as Pi
// throws.
func (t *Tree) BranchSummary(from, summary string, details jsontext.Value) (Entry, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	e := Entry{Type: TypeBranchSummary, Summary: summary, Details: details, FromID: "root"}
	if t.leaf >= 0 {
		e.FromID = t.leafID()
	}
	if from != "" {
		if _, ok := t.byID[from]; !ok {
			return Entry{}, fmt.Errorf("session: entry %s not found", from)
		}
		e.ParentID = new(from)
	}
	e.ID = t.newID(nil)
	return e, nil
}

// Labels resolve the label entries in file order, as Pi's label index does:
// each target's latest label and its timestamp, an empty or absent label
// clearing it, the targets in the order their label was first set since it
// was last cleared.
func (t *Tree) Labels() []Label {
	t.mu.Lock()
	defer t.mu.Unlock()
	return labels(t.entries)
}

func labels(entries []Entry) []Label {
	var out []Label
	for _, e := range entries {
		if e.Type != TypeLabel {
			continue
		}
		at := slices.IndexFunc(out, func(l Label) bool { return l.TargetID == e.TargetID })
		switch {
		case e.Label == nil || *e.Label == "":
			if at >= 0 {
				out = slices.Delete(out, at, at+1)
			}
		case at >= 0:
			out[at].Label, out[at].Timestamp = *e.Label, e.Timestamp
		default:
			out = append(out, Label{TargetID: e.TargetID, Label: *e.Label, Timestamp: e.Timestamp})
		}
	}
	return out
}

// Branched is the path to the entry leaf as a new session's entries, Pi's
// createBranchedSession (session-manager.ts:1632-1712): the path without
// its label entries, each entry's parent the entry kept before it; a
// compaction's firstKeptEntryId that named a label moved to the entry kept
// after it; then one label entry for each label whose target is on that
// path, chained, with the label's timestamp and a new id. An unknown leaf
// is nil.
func (t *Tree) Branched(leaf string) []Entry {
	t.mu.Lock()
	defer t.mu.Unlock()
	start, ok := t.byID[leaf]
	if !ok {
		return nil
	}
	var kept []Entry
	moved := map[string]string{} // a label's id → the entry kept after it
	var pending []string
	var parent *string
	for _, e := range pathFrom(t.entries, t.byID, start) {
		if e.Type == TypeLabel {
			pending = append(pending, e.ID)
			continue
		}
		for _, id := range pending {
			moved[id] = e.ID
		}
		pending = pending[:0]
		e.ParentID = parent
		if e.Type == TypeCompaction && e.FirstKeptEntryID != e.ID {
			if to, ok := moved[e.FirstKeptEntryID]; ok {
				e.FirstKeptEntryID = to
			}
		}
		kept = append(kept, e)
		parent = new(e.ID)
	}
	onPath := make(map[string]bool, len(kept))
	for _, e := range kept {
		onPath[e.ID] = true
	}
	taken := maps.Clone(onPath)
	for _, l := range labels(t.entries) {
		if !onPath[l.TargetID] {
			continue
		}
		e := Entry{Type: TypeLabel, ID: t.newID(taken), ParentID: parent, Timestamp: l.Timestamp, TargetID: l.TargetID, Label: new(l.Label)}
		taken[e.ID] = true
		kept = append(kept, e)
		parent = new(e.ID)
	}
	return kept
}

// newID is an id not in taken, or, when taken is nil, not in the tree.
func (t *Tree) newID(taken map[string]bool) string {
	return NewEntryID(func(id string) bool {
		if taken != nil {
			return taken[id]
		}
		_, ok := t.byID[id]
		return ok
	})
}
