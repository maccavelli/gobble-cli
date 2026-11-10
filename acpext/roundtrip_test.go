package acpext_test

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpext"
)

// frame is one recorded frame in testdata: what gobble sent or, for a
// params frame, what it decodes, and the acpext type it is.
type frame struct {
	method string     // the method it belongs to, or "" for error data
	value  func() any // a new value of its type
}

func of[T any]() func() any { return func() any { return new(T) } }

// frames are the testdata files (0042-PLAN P1 step 4, deviation 1). The
// results, notices, errors and the advertisement were captured from
// acpserver; the params were written from its JSON tags.
var frames = map[string]frame{
	"capabilities.meta.json": {"initialize", of[acpext.CapabilitiesMeta]()},

	"session.params.json":          {acpext.MethodClone, of[acpext.SessionParams]()},
	"message.params.json":          {acpext.MethodSteer, of[acpext.MessageParams]()},
	"mode.params.json":             {acpext.MethodSetSteeringMode, of[acpext.ModeParams]()},
	"compact.params.json":          {acpext.MethodCompact, of[acpext.CompactParams]()},
	"enabled.params.json":          {acpext.MethodSetAutoRetry, of[acpext.EnabledParams]()},
	"set_session_name.params.json": {acpext.MethodSetSessionName, of[acpext.SetSessionNameParams]()},
	"fork.params.json":             {acpext.MethodFork, of[acpext.ForkParams]()},
	"get_entries.params.json":      {acpext.MethodGetEntries, of[acpext.GetEntriesParams]()},
	"set_personality.params.json":  {acpext.MethodSetPersonality, of[acpext.SetPersonalityParams]()},

	"steer.result.json":                   {acpext.MethodSteer, of[acpext.QueuedResult]()},
	"follow_up.result.json":               {acpext.MethodFollowUp, of[acpext.QueuedResult]()},
	"clear_queue.result.json":             {acpext.MethodClearQueue, of[acpext.ClearedResult]()},
	"compact.result.json":                 {acpext.MethodCompact, of[acpext.CompactResult]()},
	"usage.result.json":                   {acpext.MethodUsage, of[acpext.UsageResult]()},
	"fork.result.json":                    {acpext.MethodFork, of[acpext.ForkResult]()},
	"clone.result.json":                   {acpext.MethodClone, of[acpext.CloneResult]()},
	"get_state.result.json":               {acpext.MethodGetState, of[acpext.StateResult]()},
	"get_messages.result.json":            {acpext.MethodGetMessages, of[acpext.MessagesResult]()},
	"get_entries.result.json":             {acpext.MethodGetEntries, of[acpext.EntriesResult]()},
	"get_last_assistant_text.result.json": {acpext.MethodGetLastAssistantText, of[acpext.TextResult]()},
	"get_session_stats.result.json":       {acpext.MethodGetSessionStats, of[acpext.SessionStats]()},
	"get_fork_messages.result.json":       {acpext.MethodGetForkMessages, of[acpext.ForkMessagesResult]()},
	"set_personality.result.json":         {acpext.MethodSetPersonality, of[acpext.PersonalityResult]()},

	"retry.start.notice.json":       {acpext.MethodRetry, of[acpext.RetryNotice]()},
	"retry.end.notice.json":         {acpext.MethodRetry, of[acpext.RetryNotice]()},
	"compaction.start.notice.json":  {acpext.MethodCompaction, of[acpext.CompactionNotice]()},
	"compaction.end.notice.json":    {acpext.MethodCompaction, of[acpext.CompactionNotice]()},
	"compaction.paused.notice.json": {acpext.MethodCompaction, of[acpext.CompactionNotice]()},
	"personality.notice.json":       {acpext.MethodPersonality, of[acpext.PersonalityNotice]()},

	// A provider retry's start with what the provider said (0049-MADR D2);
	// written from the JSON tags, as 0049-PLAN P3 sends it.
	"retry.start.provider.notice.json": {acpext.MethodRetry, of[acpext.RetryNotice]()},

	// CompactionMeta rides in ACP's compaction_update, under _meta.gobble
	// (0016-MADR D9, P3-Q3); written from its JSON tags, as P3c sends it.
	"compaction.meta.start.json":  {"", of[acpext.CompactionMeta]()},
	"compaction.meta.end.json":    {"", of[acpext.CompactionMeta]()},
	"compaction.meta.paused.json": {"", of[acpext.CompactionMeta]()},

	"authenticate.error.json":   {"", of[acpext.ErrorData]()},
	"prompt-auth.error.json":    {"", of[acpext.ErrorData]()},
	"invalid-params.error.json": {"", of[acpext.ErrorData]()},
	"mid-turn.error.json":       {"", of[acpext.ErrorData]()},

	// A failed turn's data (0049-MADR D1, D7), written from the JSON tags, as
	// 0049-PLAN P3 sends it: a free-tier limit, a wait over the cap, and a
	// failure after retries.
	"prompt-quota.error.json":         {"", of[acpext.ErrorData]()},
	"prompt-rate-over-cap.error.json": {"", of[acpext.ErrorData]()},
	"prompt-after-retries.error.json": {"", of[acpext.ErrorData]()},
}

// emptyResults are the requests whose result is {}, which has no type.
var emptyResults = []string{
	acpext.MethodSetSteeringMode, acpext.MethodSetFollowUpMode, acpext.MethodSetSessionName,
	acpext.MethodSetAutoRetry, acpext.MethodAbortRetry, acpext.MethodSetAutoCompaction,
}

// codec is one of the two encoders a client may use. Each decode rejects a
// member its type lacks, as magic-cli-remote's fixture test does (its
// 0179-MADR D11 item 9), so a frame with a member acpext does not name
// fails here first.
type codec struct {
	name   string
	decode func([]byte, any) error
	encode func(any) ([]byte, error)
}

var codecs = []codec{
	{"encoding/json", func(b []byte, v any) error {
		d := jsonv1.NewDecoder(bytes.NewReader(b))
		d.DisallowUnknownFields()
		return d.Decode(v)
	}, jsonv1.Marshal},
	{"encoding/json/v2", func(b []byte, v any) error {
		return jsonv2.Unmarshal(b, v, jsonv2.RejectUnknownMembers(true))
	}, func(v any) ([]byte, error) { return jsonv2.Marshal(v) }},
}

func canonical(t *testing.T, b []byte) []byte {
	t.Helper()
	v := jsontext.Value(slices.Clone(b))
	if err := v.Canonicalize(); err != nil {
		t.Fatalf("canonicalize %s: %v", b, err)
	}
	return v
}

// Every recorded frame decodes into its acpext type, with no member left
// over, and encodes back to the same JSON, compared canonically, under both
// encoders (0042-MADR Confirmation 8).
func TestFramesRoundTrip(t *testing.T) {
	for name, f := range frames {
		raw, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		want := canonical(t, raw)
		for _, c := range codecs {
			v := f.value()
			if err := c.decode(raw, v); err != nil {
				t.Errorf("%s, %s: decode: %v", name, c.name, err)
				continue
			}
			b, err := c.encode(v)
			if err != nil {
				t.Errorf("%s, %s: encode: %v", name, c.name, err)
				continue
			}
			if got := canonical(t, b); !bytes.Equal(got, want) {
				t.Errorf("%s, %s: encodes back as\n%s\nwant\n%s", name, c.name, got, want)
			}
		}
	}
}

// The frames are the testdata files, every one, and they cover every
// method: each request's result or its {}, each notification, and the
// advertisement.
func TestFramesCoverEveryMethod(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if _, ok := frames[filepath.Base(f)]; !ok {
			t.Errorf("%s has no entry in frames", f)
		}
	}
	if len(files) != len(frames) {
		t.Errorf("%d testdata files, %d frames", len(files), len(frames))
	}
	covered := map[string]bool{}
	for name, f := range frames {
		if strings.HasSuffix(name, ".result.json") || strings.HasSuffix(name, ".notice.json") {
			covered[f.method] = true
		}
	}
	for _, m := range emptyResults {
		covered[m] = true
	}
	for _, m := range slices.Concat(acpext.Requests(), acpext.Notifications()) {
		if !covered[m] {
			t.Errorf("%s has no recorded result or notice", m)
		}
	}
}

// The advertisement recorded from initialize is Requests.
func TestCapabilitiesMetaIsRequests(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "capabilities.meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta acpext.CapabilitiesMeta
	if err := jsonv2.Unmarshal(raw, &meta, jsonv2.RejectUnknownMembers(true)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(meta.Extensions, acpext.Requests()) {
		t.Errorf("initialize advertised %q, Requests %q", meta.Extensions, acpext.Requests())
	}
}

// ErrorData's members are written in sorted order, as gobble writes the
// maps it builds today, so data built from it has the same bytes.
func TestErrorDataSortedMembers(t *testing.T) {
	retryable := false
	b, err := jsonv2.Marshal(acpext.ErrorData{Attempts: 3, Code: "c", Command: "c", Error: "e", HTTPStatus: 429,
		Kind: acpext.KindQuota, Limit: acpext.LimitFreeTier, Provider: "p", ProviderMessage: "m", Reason: "r",
		ResetAt: "2026-11-09T14:05:00Z", RetryAfterMs: 5601000, Retryable: &retryable, SessionID: "s"})
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"attempts":3,"code":"c","command":"c","error":"e","httpStatus":429,"kind":"quota","limit":"free_tier",` +
		`"provider":"p","providerMessage":"m","reason":"r","resetAt":"2026-11-09T14:05:00Z","retryAfterMs":5601000,` +
		`"retryable":false,"sessionId":"s"}`
	if string(b) != want {
		t.Errorf("ErrorData encodes as %s, want %s", b, want)
	}
}

// The kinds and the limit scopes are 0049-MADR's names, D3's and D1's, so
// that gobble's own mapping can be held to them.
func TestKindsAndLimits(t *testing.T) {
	kinds := []acpext.ErrorKind{acpext.KindWorkspace, acpext.KindQuota, acpext.KindRateLimited,
		acpext.KindContextOverflow, acpext.KindEmptyReply, acpext.KindIncomplete, acpext.KindAuth,
		acpext.KindNotPermitted, acpext.KindOffline, acpext.KindUnavailable, acpext.KindUnsupported,
		acpext.KindTimeout, acpext.KindCanceled, acpext.KindOther}
	wantKinds := []string{"workspace", "quota", "rate-limited", "context-overflow", "empty_reply", "incomplete", "auth",
		"not-permitted", "offline", "unavailable", "unsupported", "timeout", "canceled", "other"}
	for i, k := range kinds {
		if string(k) != wantKinds[i] {
			t.Errorf("kind %d is %q, want %q", i, k, wantKinds[i])
		}
	}
	limits := []acpext.LimitScope{acpext.LimitRate, acpext.LimitWindow, acpext.LimitDaily, acpext.LimitWeekly,
		acpext.LimitMonthly, acpext.LimitCredits, acpext.LimitFreeTier, acpext.LimitPlan}
	wantLimits := []string{"rate", "window", "daily", "weekly", "monthly", "credits", "free_tier", "plan"}
	for i, l := range limits {
		if string(l) != wantLimits[i] {
			t.Errorf("limit %d is %q, want %q", i, l, wantLimits[i])
		}
	}
	// Where a kind is also a retry cause, the name is the same (0049-MADR D3).
	for _, c := range []acpext.RetryCause{acpext.RetryOffline, acpext.RetryEmptyReply, acpext.RetryWorkspace,
		acpext.RetryQuota, acpext.RetryRateLimited, acpext.RetryAuth, acpext.RetryNotPermitted,
		acpext.RetryUnavailable, acpext.RetryTimeout, acpext.RetryCanceled, acpext.RetryOther} {
		if !slices.Contains(kinds, acpext.ErrorKind(c)) {
			t.Errorf("retry cause %q has no kind of the same name", c)
		}
	}
}
