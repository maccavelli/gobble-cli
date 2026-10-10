package session

import (
	"encoding/json/jsontext"
	"strings"
	"testing"
)

func compaction(index string) Entry {
	return Entry{Type: TypeCompaction, Summary: "s", Unknown: jsontext.Value(`{"firstKeptEntryIndex":` + index + `,"kept":true}`)}
}

// v1's entries get fixed ids, chained in order, and a compaction's index,
// which counts the header as 0, becomes the id of that entry; an index at
// the header or past the end leaves none, as Pi leaves none
// (session-manager.ts:287-313).
func TestMigrateV1(t *testing.T) {
	user := Entry{Type: TypeMessage, Message: &Message{Role: RoleUser, Content: jsontext.Value(`"hi"`)}}
	for index, want := range map[string]string{"3": "00000003", "1": "00000001", "0": "", "4": "", "2.5": "", `"3"`: ""} {
		in := []Entry{user, user, compaction(index)}
		out, err := Migrate(0, in)
		if err != nil {
			t.Fatalf("index %s: %v", index, err)
		}
		var ids []string
		for i, e := range out {
			ids = append(ids, e.ID)
			switch {
			case i == 0 && e.ParentID != nil:
				t.Errorf("index %s: the first entry has parent %q", index, *e.ParentID)
			case i > 0 && (e.ParentID == nil || *e.ParentID != out[i-1].ID):
				t.Errorf("index %s: entry %d's parent is %v, want %s", index, i, e.ParentID, out[i-1].ID)
			}
		}
		if got := strings.Join(ids, ","); got != "00000001,00000002,00000003" {
			t.Errorf("index %s: ids %s", index, got)
		}
		c := out[2]
		if c.FirstKeptEntryID != want || string(c.Unknown) != `{"kept":true}` {
			t.Errorf("index %s: firstKeptEntryId %q, unknown %s; want %q and the index removed", index, c.FirstKeptEntryID, c.Unknown, want)
		}
		if in[2].FirstKeptEntryID != "" || !strings.Contains(string(in[2].Unknown), "firstKeptEntryIndex") || in[0].ID != "" {
			t.Errorf("index %s: the input was changed: %+v", index, in)
		}
	}
	// The same entries give the same ids at every load.
	a, _ := Migrate(1, []Entry{user, user})
	b, _ := Migrate(1, []Entry{user, user})
	if a[1].ID != b[1].ID || *a[1].ParentID != *b[1].ParentID {
		t.Fatalf("ids differ between loads: %+v %+v", a, b)
	}
}

// v2's hookMessage role becomes custom (session-manager.ts:316-330), in
// v1 files too, and other roles and versions are left alone.
func TestMigrateV2(t *testing.T) {
	hook := &Message{Role: "hookMessage", CustomType: "h", Content: jsontext.Value(`"from a hook"`)}
	in := []Entry{{Type: TypeMessage, ID: "a", Message: hook}, {Type: TypeMessage, ID: "b", Message: &Message{Role: RoleUser}}}
	for _, v := range []int{1, 2} {
		out, err := Migrate(v, in)
		if err != nil {
			t.Fatal(err)
		}
		if out[0].Message.Role != RoleCustom || out[0].Message.CustomType != "h" || out[1].Message.Role != RoleUser {
			t.Errorf("v%d: roles %q, %q", v, out[0].Message.Role, out[1].Message.Role)
		}
	}
	if hook.Role != "hookMessage" {
		t.Fatal("the input message was changed")
	}
	out, err := Migrate(3, in)
	if err != nil || out[0].Message.Role != "hookMessage" || out[0].ID != "a" {
		t.Fatalf("v3 was migrated: %+v, %v", out, err)
	}
	if _, err := Migrate(4, in); err == nil || !strings.Contains(err.Error(), "unsupported session version 4") {
		t.Fatalf("v4: %v", err)
	}
}

// The name is the latest session_info in file order, trimmed; an empty or
// absent one clears it (session-manager.ts:1318-1329).
func TestName(t *testing.T) {
	info := func(name *string) Entry { return Entry{Type: TypeSessionInfo, Name: name} }
	for _, c := range []struct {
		entries []Entry
		want    string
	}{
		{nil, ""},
		{[]Entry{info(new("first")), info(new("  second \t"))}, "second"},
		{[]Entry{info(new("first")), info(new(""))}, ""},
		{[]Entry{info(new("first")), info(nil)}, ""},
		{[]Entry{info(new("first")), {Type: TypeMessage}}, "first"},
	} {
		if got := Name(c.entries); got != c.want {
			t.Errorf("Name(%+v) = %q, want %q", c.entries, got, c.want)
		}
	}
	if got := Summarize(Header{}, []Entry{info(new("a")), info(new(" "))}, "").Name; got != "" {
		t.Errorf("Summarize's name %q, want none", got)
	}
}
