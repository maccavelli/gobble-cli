package jsonx

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"maps"
	"slices"
)

// Options are the data policy's options: invalid UTF-8 written and read as
// U+FFFD, and map keys sorted. They serve a jsontext.Encoder or Decoder, or
// a v2 call that needs them.
func Options() json.Options {
	return json.JoinOptions(jsontext.AllowInvalidUTF8(true), json.Deterministic(true))
}

// StrictOptions are the configuration policy's options: encoding/json/v2's
// defaults, which refuse invalid UTF-8 and a repeated member name, with map
// keys sorted.
func StrictOptions() json.Options { return json.Deterministic(true) }

// Marshal encodes v under the data policy, then o.
func Marshal(v any, o ...json.Options) ([]byte, error) {
	return json.Marshal(v, append([]json.Options{Options()}, o...)...)
}

// Unmarshal decodes b into v under the data policy, then o.
func Unmarshal(b []byte, v any, o ...json.Options) error {
	return json.Unmarshal(b, v, append([]json.Options{Options()}, o...)...)
}

// MarshalStrict encodes v under the configuration policy, then o.
func MarshalStrict(v any, o ...json.Options) ([]byte, error) {
	return json.Marshal(v, append([]json.Options{StrictOptions()}, o...)...)
}

// UnmarshalStrict decodes b into v under the configuration policy, then o.
func UnmarshalStrict(b []byte, v any, o ...json.Options) error {
	return json.Unmarshal(b, v, append([]json.Options{StrictOptions()}, o...)...)
}

// Value is v encoded by Marshal, as a hatch value: never empty non-nil.
func Value(v any) (jsontext.Value, error) {
	b, err := Marshal(v)
	if err != nil {
		return nil, err
	}
	return From(b), nil
}

// Quote is s as a JSON string literal, each invalid UTF-8 byte written as
// U+FFFD. jsontext.AppendQuote's error says only that bytes were replaced,
// which is the data policy, so Quote has none: its output equals Marshal's
// byte for byte (0039-MADR, measured over 200,009 inputs; FuzzQuote).
func Quote(s string) jsontext.Value {
	b, _ := jsontext.AppendQuote(nil, s) //nolint:errcheck // it reports a replacement, which the data policy makes
	return b
}

// Field is one member Fields writes.
type Field struct{ Name, Value string }

// Fields is a JSON object of the fields whose Value is not empty, in the
// order given, each name and value written by Quote; {} when none is. That
// is what Marshal writes for a struct of omitzero string fields in that
// order, and it cannot fail (FuzzFields).
func Fields(fs ...Field) jsontext.Value {
	var b bytes.Buffer
	b.WriteByte('{')
	first := true
	for _, f := range fs {
		if f.Value == "" {
			continue
		}
		if !first {
			b.WriteByte(',')
		}
		first = false
		b.Write(Quote(f.Name))
		b.WriteByte(':')
		b.Write(Quote(f.Value))
	}
	b.WriteByte('}')
	return b.Bytes()
}

// Strings is a JSON object of m, its keys sorted as Marshal sorts them, each
// key and value written by Quote; {} when m is empty. That is what Marshal
// writes for m, and it cannot fail: a key that Quote makes the same name as
// one already written, two invalid bytes repaired alike, is left out, where
// Marshal refuses the repeated name (0039-MADR D14; FuzzStrings).
func Strings(m map[string]string) jsontext.Value {
	var b bytes.Buffer
	b.WriteByte('{')
	written := make(map[string]bool, len(m))
	for _, k := range slices.Sorted(maps.Keys(m)) {
		name := Quote(k)
		if written[string(name)] {
			continue
		}
		if len(written) > 0 {
			b.WriteByte(',')
		}
		written[string(name)] = true
		b.Write(name)
		b.WriteByte(':')
		b.Write(Quote(m[k]))
	}
	b.WriteByte('}')
	return b.Bytes()
}

// SetMember is a copy of meta, or a new map when meta is nil, with key set
// to v encoded by Value: how gobble writes its member of an ACP _meta,
// which the SDK splices (0039-MADR D15).
func SetMember(meta map[string]any, key string, v any) (map[string]any, error) {
	b, err := Value(v)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(meta)+1)
	maps.Copy(out, meta)
	out[key] = b
	return out, nil
}

// Member is meta[key] encoded by Marshal and decoded into T by Unmarshal:
// how gobble reads a member of an ACP _meta the SDK has decoded. It is
// false when the member is missing, does not encode, or does not decode
// into T, so a mistyped member reads as none (0039-MADR D15).
func Member[T any](meta map[string]any, key string) (T, bool) {
	var t T
	v, ok := meta[key]
	if !ok {
		return t, false
	}
	b, err := Marshal(v)
	if err != nil {
		return t, false
	}
	if err := Unmarshal(b, &t); err != nil {
		var zero T
		return zero, false
	}
	return t, true
}

// From is b as a hatch value: nil or empty is absent (nil). It does not
// validate; a parent's marshal reports bad JSON.
func From(b []byte) jsontext.Value {
	if len(b) == 0 {
		return nil
	}
	return jsontext.Value(b)
}

// Splice is v as it may be spliced into another encoder, such as an ACP
// update's rawInput: absent when empty; v when it is JSON, invalid UTF-8
// allowed, since the encoder repairs that; otherwise v's text as a JSON
// string, so that the update is still sent (0039-MADR D9).
func Splice(v jsontext.Value) jsontext.Value {
	switch {
	case len(v) == 0:
		return nil
	case v.IsValid(jsontext.AllowInvalidUTF8(true)):
		return v
	}
	return Quote(string(v))
}

// IsAbsent reports a missing value: len(v) == 0. A JSON null is present.
func IsAbsent(v jsontext.Value) bool { return len(v) == 0 }

// IsNull reports a JSON null, spaces around it trimmed. Absent is not null.
func IsNull(v jsontext.Value) bool { return bytes.Equal(bytes.TrimSpace(v), []byte("null")) }
