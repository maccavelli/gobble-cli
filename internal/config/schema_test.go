package config

import (
	"bytes"
	"os"
	"testing"
)

// The committed schema is the generated one (codex's drift test, 0018-MADR
// D12): a key changed without regenerating it fails here.
func TestSchemaCommitted(t *testing.T) {
	got, err := Schema()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/settings.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("the committed schema is stale: run go run ./internal/config/genschema")
	}
}
