package acpext_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpext"
)

// Requests is the frozen table gobble advertises: 0002-PLAN Phase 7's
// sixteen, 0005-PLAN F5-1's two, F5-2's one and 0022-PLAN P3's one, sorted.
func TestRequests(t *testing.T) {
	frozen := []string{
		"_gobble/abort_retry", "_gobble/clear_queue", "_gobble/clone", "_gobble/compact", "_gobble/follow_up", "_gobble/fork",
		"_gobble/get_entries", "_gobble/get_fork_messages", "_gobble/get_last_assistant_text", "_gobble/get_messages",
		"_gobble/get_session_stats", "_gobble/get_state", "_gobble/set_auto_compaction", "_gobble/set_auto_retry", "_gobble/set_follow_up_mode",
		"_gobble/set_personality", "_gobble/set_session_name", "_gobble/set_steering_mode", "_gobble/steer", "_gobble/usage",
	}
	if got := acpext.Requests(); !slices.Equal(got, frozen) {
		t.Errorf("Requests() = %q, want the frozen %q", got, frozen)
	}
}

// Notifications are gobble's three, sorted.
func TestNotifications(t *testing.T) {
	want := []string{"_gobble/compaction", "_gobble/personality", "_gobble/retry"}
	if got := acpext.Notifications(); !slices.Equal(got, want) {
		t.Errorf("Notifications() = %q, want %q", got, want)
	}
}

// Both lists are sorted, distinct, disjoint and in gobble's namespace, and
// each call returns a slice of its own.
func TestNameLists(t *testing.T) {
	for _, list := range [][]string{acpext.Requests(), acpext.Notifications()} {
		if !slices.IsSorted(list) || len(slices.Compact(slices.Clone(list))) != len(list) {
			t.Errorf("%q is not sorted and distinct", list)
		}
		for _, m := range list {
			if !strings.HasPrefix(m, "_gobble/") {
				t.Errorf("%q is outside _gobble/", m)
			}
		}
	}
	for _, m := range acpext.Notifications() {
		if slices.Contains(acpext.Requests(), m) {
			t.Errorf("%s is both a request and a notification", m)
		}
	}
	r := acpext.Requests()
	r[0] = "changed"
	if acpext.Requests()[0] == "changed" {
		t.Error("Requests() returned a shared slice")
	}
}
