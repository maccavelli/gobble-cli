package session

import (
	"bytes"
	"testing"

	"encoding/json/jsontext"
	json "encoding/json/v2"
)

func TestEntryUnknownFieldRoundTrip(t *testing.T) {
	in := []byte(`{"type":"message","id":"abc","extra":"kept","zzz":true}`)

	var entry Entry
	if err := json.Unmarshal(in, &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if entry.Type != "message" || entry.ID != "abc" {
		t.Fatalf("known fields = %#v", entry)
	}

	out, err := json.Marshal(&entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got, want map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("remarshal decode: %v", err)
	}
	if err := json.Unmarshal(in, &want); err != nil {
		t.Fatalf("input decode: %v", err)
	}
	if got["extra"] != want["extra"] || got["zzz"] != want["zzz"] {
		t.Fatalf("unknown fields not preserved\n got %s\nwant members extra and zzz from %s", out, in)
	}
	if _, err := jsontext.NewDecoder(bytes.NewReader(out)).ReadValue(); err != nil {
		t.Fatalf("output is not one JSON value: %v", err)
	}
}
