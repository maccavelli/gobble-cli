// Package jsonx is gobble's one way to encode and decode JSON, on
// encoding/json/v2, under two policies (0039-MADR D1 to D3).
// Stability: stable
//
// The data policy, Marshal and Unmarshal, is for protocol, session, tool
// and provider data and CLI output. An invalid UTF-8 byte in a string is
// written, and read, as U+FFFD, one for each byte, as encoding/json v1 and
// the SDKs gobble uses write it; a lone surrogate escape reads as U+FFFD;
// a map is written with its keys sorted. So no text gobble reads can make
// an encode fail, and what gobble saves is what the model saw.
//
// The configuration policy, MarshalStrict and UnmarshalStrict, is for the
// configuration and state files. It keeps encoding/json/v2's defaults,
// which refuse invalid UTF-8 and a repeated object member name, and sorts
// map keys. A malformed file is refused with its error, never repaired.
//
// Raw JSON is a jsontext.Value; binary is a []byte, which encodes as
// base64. A jsontext.Value that is empty is absent, which From, Value and
// Splice keep so: an empty non-nil value on an omitzero field fails a
// marshal. jsontext.Value's String gives "null" for an absent value, so a
// test or a log compares len and string(v), never String.
//
// jsonx imports no other package of this module.
package jsonx
