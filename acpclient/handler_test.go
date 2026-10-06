package acpclient

import (
	"testing"

	acp "github.com/coder/acp-go-sdk"
)

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
