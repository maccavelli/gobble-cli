package acpclient

import (
	"context"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/jsonx"
)

// A permission request's _meta.gobble is read one member at a time: a
// mistyped repeated is none, and the limit beside it is still read
// (0039-MADR D15, Confirmation 21).
func TestMetaMemberMistyped(t *testing.T) {
	var asked PermissionRequest
	h := &handler{permission: func(_ context.Context, r PermissionRequest) PermissionAnswer {
		asked = r
		return PermissionAnswer{}
	}}
	req := acp.RequestPermissionRequest{Options: []acp.PermissionOption{{OptionId: "n", Name: "n", Kind: acp.PermissionOptionKindRejectOnce}},
		Meta: map[string]any{"gobble": map[string]any{"repeated": "3", "limit": "a delete always asks"}}}
	if _, err := h.RequestPermission(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if asked.Limit != "a delete always asks" || asked.Repeated != 0 {
		t.Fatalf("asked %+v, want the limit and no repeated", asked)
	}
}

// The CLI declines every permission request until Phase 6's prompt: the
// first reject_once option, else reject_always, else cancelled.
func TestPermissionDeclined(t *testing.T) {
	opt := func(id string, k acp.PermissionOptionKind) acp.PermissionOption {
		return acp.PermissionOption{OptionId: acp.PermissionOptionId(id), Name: id, Kind: k}
	}
	cases := []struct {
		name string
		opts []acp.PermissionOption
		want string
	}{
		{"reject once", []acp.PermissionOption{opt("a", acp.PermissionOptionKindAllowOnce), opt("r", acp.PermissionOptionKindRejectOnce), opt("R", acp.PermissionOptionKindRejectAlways)}, "r"},
		{"reject always", []acp.PermissionOption{opt("a", acp.PermissionOptionKindAllowOnce), opt("R", acp.PermissionOptionKindRejectAlways)}, "R"},
		{"no reject", []acp.PermissionOption{opt("a", acp.PermissionOptionKindAllowOnce)}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := (&handler{}).RequestPermission(t.Context(), acp.RequestPermissionRequest{Options: tc.opts})
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case tc.want == "" && resp.Outcome.Cancelled == nil:
				t.Fatalf("outcome %+v, want cancelled", resp.Outcome)
			case tc.want != "" && (resp.Outcome.Selected == nil || string(resp.Outcome.Selected.OptionId) != tc.want):
				t.Fatalf("outcome %+v, want %s selected", resp.Outcome, tc.want)
			}
		})
	}
}

// Each of the client's _meta.gobble writers writes the bytes v1 wrote from
// the map it replaces, whose keys v1 sorted (0039-MADR D15; 0039-PLAN
// deviation 8).
func TestMetaWireBytes(t *testing.T) {
	bytesOf := func(meta map[string]any, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		b, err := jsonx.Marshal(meta)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	block, err := unscanned("ctx")
	if err != nil {
		t.Fatal(err)
	}
	h := &handler{permission: func(context.Context, PermissionRequest) PermissionAnswer {
		return PermissionAnswer{OptionID: "n", Feedback: "use git mv"}
	}}
	resp, err := h.RequestPermission(t.Context(), acp.RequestPermissionRequest{Options: []acp.PermissionOption{{OptionId: "n", Name: "n", Kind: acp.PermissionOptionKindRejectOnce}}})
	for _, tc := range []struct {
		name, got, want string
	}{
		{"session/new", bytesOf(NewSessionOptions{ID: "i", ForkFrom: "f", Name: "n"}.meta()), `{"gobble":{"forkFrom":"f","name":"n","sessionId":"i"}}`},
		{"mentions", bytesOf(block.Text.Meta, nil), `{"gobble":{"mentions":false}}`},
		{"feedback", bytesOf(resp.Meta, err), `{"gobble":{"feedback":"use git mv"}}`},
	} {
		if tc.got != tc.want {
			t.Errorf("%s _meta %s, want %s", tc.name, tc.got, tc.want)
		}
	}
}
