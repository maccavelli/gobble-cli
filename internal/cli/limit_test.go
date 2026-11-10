package cli

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/jsonx"
)

// quotaFrame is acpext's prompt-quota.error.json: the data of a turn
// refused on OpenCode Zen's free tier, which lifts at 14:05 UTC.
func quotaFrame(t *testing.T) (string, acpext.ErrorData) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "acpext", "testdata", "prompt-quota.error.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d acpext.ErrorData
	if err := jsonx.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return string(b), d
}

// The limit line names the limit in words and the provider, and when the
// limit lifts on the local clock with the wait left (0049-MADR Q3 (a)).
func TestLimitLine(t *testing.T) {
	cst := time.FixedZone("CST", -6*60*60)
	now := time.Date(2026, 11, 9, 7, 0, 0, 0, cst) // 13:00 UTC
	for _, c := range []struct {
		name string
		d    acpext.ErrorData
		want string
	}{
		{"free tier", acpext.ErrorData{Limit: acpext.LimitFreeTier, Provider: "opencode-zen", ResetAt: "2026-11-09T14:05:00Z"},
			"Free tier limit on opencode-zen; resets at 08:05, in 1h5m"},
		{"under a minute", acpext.ErrorData{Limit: acpext.LimitRate, Provider: "openai", ResetAt: "2026-11-09T13:00:40Z"},
			"Rate limit on openai; resets at 07:00, in 40s"},
		{"passed", acpext.ErrorData{Limit: acpext.LimitDaily, Provider: "gemini", ResetAt: "2026-11-09T12:00:00Z"},
			"Daily quota on gemini; resets at 06:00"},
		{"no reset", acpext.ErrorData{Limit: acpext.LimitCredits, Provider: "kilo"}, "Credits used up on kilo"},
		{"no limit", acpext.ErrorData{Provider: "claude", ResetAt: "2026-11-09T15:00:00Z"}, "Limit on claude; resets at 09:00, in 2h"},
		{"a limit not in the table", acpext.ErrorData{Limit: "hourly"}, "hourly"},
		{"neither", acpext.ErrorData{Provider: "claude", Kind: acpext.KindOffline}, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := limitLine(c.d, now)
			if got != c.want || ok != (c.want != "") {
				t.Fatalf("limitLine = %q %v, want %q", got, ok, c.want)
			}
		})
	}
	for _, s := range []acpext.LimitScope{acpext.LimitRate, acpext.LimitWindow, acpext.LimitDaily, acpext.LimitWeekly,
		acpext.LimitMonthly, acpext.LimitCredits, acpext.LimitFreeTier, acpext.LimitPlan} {
		if limitWords[s] == "" {
			t.Errorf("no words for %q", s)
		}
	}
}

// A turn refused on a limit prints the agent's message, not the JSON-RPC
// error, and the limit line under it; the session goes on.
func TestLineSessionLimitError(t *testing.T) {
	data, d := quotaFrame(t)
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{ErrCode: -32603, ErrMessage: "Internal error", ErrData: data},
		{Updates: []string{chunk("fine")}}}}
	h := newLineHarness(t, agent)
	h.e.now = func() time.Time { return time.Date(2026, 11, 9, 13, 0, 0, 0, time.UTC) }
	h.start(Prompt{})
	h.press("one\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "resets at") })
	h.press("two\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "fine") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	got := h.stderr.String()
	if !strings.Contains(got, "Error: "+d.Error+"\n") || !strings.Contains(got, "Free tier limit on opencode-zen; resets at 14:05, in 1h5m\n") ||
		strings.Contains(got, "session/prompt:") || strings.Contains(got, `"code"`) {
		t.Fatalf("stderr %q", got)
	}
}

// Print mode's JSON result carries the agent's message as error and its
// data as errorData (0049-MADR Q3 (a)).
func TestPrintErrorData(t *testing.T) {
	data, d := quotaFrame(t)
	withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{{ErrCode: -32603, ErrMessage: "Internal error", ErrData: data}}})
	r := runMain(t, nil, "-p", "--output-format", "json", "x")
	if r.code != ExitFailure {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	var res struct {
		Error     string         `json:"error"`
		ErrorData jsontext.Value `json:"errorData"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(r.stdout)), &res); err != nil {
		t.Fatalf("stdout %q: %v", r.stdout, err)
	}
	var got acpext.ErrorData
	if err := jsonx.UnmarshalStrict(res.ErrorData, &got); err != nil {
		t.Fatalf("errorData %s: %v", res.ErrorData, err)
	}
	gb, _ := jsonx.Marshal(got) //nolint:errcheck // a decoded ErrorData encodes
	wb, _ := jsonx.Marshal(d)   //nolint:errcheck // as above
	if res.Error != d.Error || string(gb) != string(wb) {
		t.Fatalf("result error %q, errorData %s; want %q and %s", res.Error, gb, d.Error, wb)
	}
}
