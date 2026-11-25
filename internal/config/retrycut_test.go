package config

import (
	"testing"
)

// retry.retryAfterCut is a boolean session setting beside the retry keys,
// on by default, with the record's doc; a file turns it off (0070-MADR D12
// (f), C19, C20).
func TestRetryAfterCutSetting(t *testing.T) {
	const doc = "Whether a model call cut by a dropped connection, an idle timeout or a server error after its reply began is sent again from the start, at most 3 times in a row; the cut part stays in the session as a failed attempt."
	k, ok := KeyOf("retry.retryAfterCut")
	if !ok || k.Kind != KindBool || k.Default != "true" || k.Read != ReadSession || k.Doc != doc {
		t.Fatalf("retry.retryAfterCut: %+v, %v; want a session bool, default true, with the record's doc", k, ok)
	}
	r, ds := Resolve(Options{})
	if v, ok := r.Settings.Value("retry.retryAfterCut"); len(ds) != 0 || !ok || v != true {
		t.Fatalf("default retry.retryAfterCut = %v, %v (%v), want true", v, ok, ds)
	}
	l := newLayout(t, "retry:\n  retryAfterCut: false\n", "")
	r, ds = Resolve(Options{ConfigDir: l.config, Cwd: l.project, Trusted: trustAll})
	if v, ok := r.Settings.Value("retry.retryAfterCut"); len(ds) != 0 || !ok || v != false {
		t.Fatalf("retry.retryAfterCut from a file = %v, %v (%v), want false", v, ok, ds)
	}
}
