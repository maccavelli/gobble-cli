package config

import (
	"bytes"
	"strings"
	"testing"
)

// images.autoResize is a session key, default true, in the schema and the
// template (0036-PLAN P7).
func TestImagesAutoResize(t *testing.T) {
	k, ok := KeyOf("images.autoResize")
	if !ok || k.Kind != KindBool || k.Default != boolTrue || k.Read != ReadSession {
		t.Fatalf("images.autoResize = %+v, present %v", k, ok)
	}
	if !Defaults().Images.AutoResize {
		t.Fatal("the default is not true")
	}
	if !bytes.Contains(Template(), []byte("# images:\n#   autoResize: true\n")) {
		t.Fatal("the template has no images.autoResize at its default")
	}
	schema, err := Schema()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(schema, []byte(`"autoResize"`)) || !strings.Contains(string(schema), `"default": true`) {
		t.Fatal("the schema has no images.autoResize defaulting to true")
	}
}
