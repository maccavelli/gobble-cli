package auth

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/appdirs"
	"github.com/maccavelli/gobble-cli/internal/filelock"
	"github.com/maccavelli/gobble-cli/jsonx"
)

// The fixtures are files of the shapes the owner's can have, built in a
// temporary directory: no test reads the owner's files (0052-PLAN C10).
// Every secret value holds "secret", so a test can see one leak.
const (
	hostA   = "urn:uuid:11111111-2222-4333-8444-555555555555"
	hostB   = "urn:uuid:aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	clientA = "app_planclient"
	dormant = `"key": "sk-dormant-secret"`
	dormWS  = `"workspace": "ws-dormant-secret"`
)

// planFields are openai's plan entry as a build of the archived work wrote
// it: no kind member (0052-MADR D9, Q6).
func planFields() []string {
	return []string{
		`"type": "oauth"`, `"access": "plan-access-secret"`, `"refresh": "plan-refresh-secret"`, `"expires": 1781000000000`,
		`"issuer": "https://auth.openai.com"`, `"clientId": "` + clientA + `"`, `"tokenUrl": "https://auth.openai.com/oauth/token"`,
		`"resource": "https://api.openai.com/v1"`, `"scope": "openid"`, `"idToken": "plan-id-token-secret"`, `"hostId": "` + hostA + `"`,
	}
}

// without is fields less those whose name is one of names.
func without(fields []string, names ...string) []string {
	return slices.DeleteFunc(slices.Clone(fields), func(f string) bool {
		return slices.ContainsFunc(names, func(n string) bool { return strings.HasPrefix(f, `"`+n+`":`) })
	})
}

// entryText is one provider's entry as gobble lays it out.
func entryText(provider string, fields ...string) string {
	return strconv.Quote(provider) + ": {\n    " + strings.Join(fields, ",\n    ") + "\n  }"
}

// planMember is openai's plan entry with extra members after its own.
func planMember(extra ...string) string {
	return entryText("openai", append(planFields(), extra...)...)
}

// authFile is a file of members, as gobble writes one.
func authFile(members ...string) string {
	if len(members) == 0 {
		return "{}\n"
	}
	return "{\n  " + strings.Join(members, ",\n  ") + "\n}\n"
}

// syncBuf is a buffer two runs may log to at once.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// mfix is a scratch config directory and state directory, with stub hooks:
// PlanSession reports openai's oauth entry as a plan session when plan is
// set, answered when its consents member is set; CarryConsent records an
// answer in that member through SetOAuth, as provider.SetConsent does, and
// first notes whether auth.json.lock is free.
type mfix struct {
	t               *testing.T
	config, state   string
	auth            string
	plan            bool
	settings        [2]bool
	carryErr        error
	noLog           bool
	now             time.Time
	settingsCalls   atomic.Int32
	planCalls       atomic.Int32
	carries         atomic.Int32
	lockFreeAtCarry atomic.Int32
	lockHeldAtCarry atomic.Int32
	log             syncBuf
}

func newMfix(t *testing.T) *mfix {
	t.Helper()
	dir := t.TempDir()
	return &mfix{t: t, config: filepath.Join(dir, "config"), state: filepath.Join(dir, "state"),
		auth: filepath.Join(dir, "config", "auth.json"), plan: true, settings: [2]bool{false, true},
		now: time.Date(2026, 11, 22, 9, 8, 7, 0, time.UTC)}
}

func (f *mfix) write(path, text string) {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		f.t.Fatal(err)
	}
}

func (f *mfix) read(path string) string {
	f.t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		f.t.Fatal(err)
	}
	return string(b)
}

func (f *mfix) opts() MigrateOptions {
	o := MigrateOptions{
		AuthPath: f.auth, StateDir: f.state,
		Settings: func() (bool, bool) {
			f.settingsCalls.Add(1)
			return f.settings[0], f.settings[1]
		},
		PlanSession: func(context.Context) (bool, bool, error) {
			f.planCalls.Add(1)
			file, err := Load(f.auth)
			if err != nil {
				return false, false, err
			}
			e, ok := file.Entry("openai")
			if !ok || e.Type != TypeOAuth || !f.plan {
				return false, false, nil
			}
			return true, e.Consents != "", nil
		},
		CarryConsent: func(ctx context.Context) error {
			f.carries.Add(1)
			if release, err := filelock.TryLock(f.auth + ".lock"); err == nil {
				f.lockFreeAtCarry.Add(1)
				if err := release(); err != nil {
					return err
				}
			} else {
				f.lockHeldAtCarry.Add(1)
			}
			if f.carryErr != nil {
				return f.carryErr
			}
			file, err := Load(f.auth)
			if err != nil {
				return err
			}
			e, _ := file.Entry("openai")
			e.Consents = "test.consent=granted"
			return SetOAuth(ctx, f.auth, "openai", e)
		},
		Now: func() time.Time { return f.now },
	}
	if !f.noLog {
		o.Log = func() *slog.Logger { return slog.New(slog.NewJSONHandler(&f.log, nil)) }
	}
	return o
}

func (f *mfix) run() (Report, error) {
	f.t.Helper()
	return Migrate(f.t.Context(), f.opts())
}

// names are the files under the config and state directories, by their
// path from the directory holding both, sorted.
func (f *mfix) names() []string {
	f.t.Helper()
	var out []string
	root := filepath.Dir(f.config)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		out = append(out, filepath.ToSlash(rel))
		return err
	})
	if err != nil {
		f.t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// backups are the names of the backups beside auth.json.
func (f *mfix) backups() []string {
	f.t.Helper()
	m, err := filepath.Glob(filepath.Join(f.config, "auth.json.bak-*"))
	if err != nil {
		f.t.Fatal(err)
	}
	return m
}

// noneOf fails when a name in names holds any of parts.
func noneOf(t *testing.T, what string, names []string, parts ...string) {
	t.Helper()
	for _, n := range names {
		for _, p := range parts {
			if strings.Contains(n, p) {
				t.Errorf("%s: %s left %s", what, p, n)
			}
		}
	}
}

// stateFiles are the files in the state directory.
func (f *mfix) stateFiles() []string {
	f.t.Helper()
	des, err := os.ReadDir(f.state)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		f.t.Fatal(err)
	}
	var out []string
	for _, de := range des {
		out = append(out, de.Name())
	}
	return out
}

// records are the log's records with message msg.
func (f *mfix) records(msg string) []map[string]any {
	f.t.Helper()
	var out []map[string]any
	for line := range strings.Lines(f.log.String()) {
		var r map[string]any
		if err := jsonx.UnmarshalStrict([]byte(line), &r); err != nil {
			f.t.Fatalf("log line %q: %v", line, err)
		}
		if r["msg"] == msg {
			out = append(out, r)
		}
	}
	return out
}

// rawMembers are a file's members' bytes as written, in its order.
func rawMembers(t *testing.T, data string) (names []string, values map[string]string) {
	t.Helper()
	values = map[string]string{}
	d := jsontext.NewDecoder(strings.NewReader(data))
	if _, err := d.ReadToken(); err != nil {
		t.Fatal(err)
	}
	for d.PeekKind() != '}' {
		tok, err := d.ReadToken()
		if err != nil {
			t.Fatal(err)
		}
		name := tok.String()
		v, err := d.ReadValue()
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
		values[name] = string(v)
	}
	return names, values
}

// fieldsOf is an entry's fields by name, each as compact JSON.
func fieldsOf(t *testing.T, raw string) map[string]string {
	t.Helper()
	var m map[string]jsontext.Value
	if err := jsonx.UnmarshalStrict([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for k, v := range m {
		c := v.Clone()
		if err := c.Compact(); err != nil {
			t.Fatal(err)
		}
		out[k] = string(c)
	}
	return out
}

// setHooks sets the migration's test hooks for t and restores them after.
func setHooks(t *testing.T, set func()) {
	t.Helper()
	old := migrateHooks
	t.Cleanup(func() { migrateHooks = old })
	set()
}

var errFault = errors.New("injected fault")

// A file with nothing to migrate is left as it is: no file, an empty or
// white-space file, an oauth entry with neither member and no plan facts.
// Nothing is written, no backup, no state file, no record, no carry
// (rows M1, M2, M6).
func TestMigrateNothingToDo(t *testing.T) {
	for name, text := range map[string]string{
		"no file":        "",
		"empty":          "\x00",
		"white space":    "  \n\t\n",
		"no dormant key": authFile(entryText("anthropic", `"type": "oauth"`, `"access": "a-secret"`, `"refresh": "r-secret"`)),
	} {
		t.Run(name, func(t *testing.T) {
			f := newMfix(t)
			f.settings = [2]bool{true, true}
			if text == "\x00" {
				f.write(f.auth, "")
			} else if text != "" {
				f.write(f.auth, text)
			}
			before := f.names()
			r, err := f.run()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(r, Report{}) {
				t.Errorf("Report = %+v, want nothing done", r)
			}
			if got := f.names(); !slices.Equal(got, before) {
				t.Errorf("files %v, want %v as before", got, before)
			}
			if text != "" && text != "\x00" && f.read(f.auth) != text {
				t.Errorf("auth.json changed")
			}
			if f.carries.Load() != 0 || f.log.String() != "" {
				t.Errorf("carries %d, log %q; want none", f.carries.Load(), f.log.String())
			}
		})
	}
}

// With nothing to migrate no lock is taken and nothing is created: not the
// config directory, not auth.json.lock (row M41).
func TestMigrateNothingToDoTakesNoLock(t *testing.T) {
	f := newMfix(t)
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.config); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the config directory exists (%v); no file was to migrate", err)
	}
	f.write(f.auth, authFile(entryText("anthropic", `"type": "api_key"`, `"key": "k-secret"`)))
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.auth + ".lock"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("auth.json.lock exists (%v); nothing was to migrate", err)
	}
}

// A file that cannot be read or parsed is an error naming auth.json, and
// nothing is written: no backup, no state file, no carry (row M3).
func TestMigrateUnreadableFileWritesNothing(t *testing.T) {
	cases := map[string]string{
		"not an object":    "[]\n",
		"no string type":   `{"openai": {"type": 1, ` + dormant + `}}` + "\n",
		"duplicate member": `{"openai": {"type": "oauth", ` + dormant + `}, "openai": {"type": "oauth"}}` + "\n",
		"invalid UTF-8":    "{\"openai\": {\"type\": \"oauth\", \"key\": \"\xff\"}}\n",
	}
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		cases["mode 000"] = authFile(planMember(dormant))
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			f := newMfix(t)
			f.settings = [2]bool{true, true}
			f.write(f.auth, text)
			if name == "mode 000" {
				if err := os.Chmod(f.auth, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(f.auth, 0o600) })
			}
			before := f.names()
			_, err := f.run()
			if err == nil || !strings.Contains(err.Error(), "auth.json") {
				t.Errorf("err = %v, want an error naming auth.json", err)
			}
			if !errors.Is(err, ErrUnreadable) { // M66
				t.Errorf("err = %v, want it marked ErrUnreadable, for the command to report", err)
			}
			if name == "mode 000" {
				if err := os.Chmod(f.auth, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if f.read(f.auth) != text {
				t.Error("auth.json changed")
			}
			if got := f.names(); !slices.Equal(got, before) {
				t.Errorf("files %v, want %v as before", got, before)
			}
			if f.carries.Load() != 0 {
				t.Error("CarryConsent was called")
			}
		})
	}
}

// A dormant key or workspace leaves each oauth entry, whatever its value and
// whether or not the provider is one gobble knows; one Info record per
// provider names it and the backup, never a value; Report names the same,
// with no logger as well (rows M4, M5, M49, M50, M60).
func TestMigrateDropsDormantKey(t *testing.T) {
	text := authFile(
		entryText("anthropic", `"type": "oauth"`, `"access": "a-secret"`, dormant, dormWS),
		entryText("somecorp", `"type": "oauth"`, `"access": "s-secret"`, `"clientId": "`+clientA+`"`, `"hostId": "`+hostA+`"`, dormant),
		entryText("xai", `"type": "oauth"`, `"access": "x-secret"`, dormant),
		entryText("kilo", `"type": "oauth"`, `"access": "k-secret"`, dormWS),
		entryText("e1", `"type": "oauth"`, `"key": ""`),
		entryText("e2", `"type": "oauth"`, `"key": null`),
		entryText("e3", `"type": "oauth"`, `"workspace": ""`),
	)
	want := []string{"anthropic", "somecorp", "xai", "kilo", "e1", "e2", "e3"}
	for _, noLog := range []bool{false, true} {
		f := newMfix(t)
		f.noLog = noLog
		f.settings = [2]bool{true, true}
		f.write(f.auth, text)
		r, err := f.run()
		if err != nil {
			t.Fatal(err)
		}
		backups := f.backups()
		if len(backups) != 1 {
			t.Fatalf("backups %v, want one", backups)
		}
		backup := filepath.Base(backups[0])
		if !slices.Equal(r.Dropped, want) || r.Backup != backup {
			t.Errorf("noLog %v: Report %+v, want dropped %v and backup %s", noLog, r, want, backup)
		}
		_, values := rawMembers(t, f.read(f.auth))
		for _, p := range want {
			got := fieldsOf(t, values[p])
			if _, ok := got["key"]; ok {
				t.Errorf("%s keeps key", p)
			}
			if _, ok := got["workspace"]; ok {
				t.Errorf("%s keeps workspace", p)
			}
		}
		if got := f.stateFiles(); len(got) != 0 || f.carries.Load() != 0 || f.planCalls.Load() != 0 {
			t.Errorf("state %v, carries %d, PlanSession calls %d; want none: no openai entry", got, f.carries.Load(), f.planCalls.Load())
		}
		if noLog {
			continue
		}
		recs := f.records("auth.json: dropped a dormant key or workspace")
		var named []string
		for _, rec := range recs {
			named = append(named, rec["provider"].(string))
			if rec["level"] != "INFO" || rec["backup"] != backup {
				t.Errorf("record %v: want level INFO naming the backup %s", rec, backup)
			}
		}
		if !slices.Equal(named, want) {
			t.Errorf("records name %v, want one per provider %v", named, want)
		}
		if strings.Contains(f.log.String(), "secret") {
			t.Errorf("the log holds a value:\n%s", f.log.String())
		}
	}
}

// Every other member and field is kept: an api_key entry's key and
// workspace, unknown providers, an entry of another type, the members'
// order, a member over several lines and one on one line, byte for byte;
// the edited entry's other fields, an unknown one included (rows M7, M8,
// M9).
func TestMigrateKeepsEverythingElse(t *testing.T) {
	text := "{\n" +
		`  "zzz-custom": {"type": "api_key", "key": "z-secret"},` + "\n" +
		"  " + entryText("anthropic", `"type": "api_key"`, `"key": "a-secret"`, `"workspace": "ws-1"`) + ",\n" +
		"  " + entryText("custom", `"type": "other"`, `"key": "c-secret"`, `"workspace": "w"`) + ",\n" +
		"  " + planMember(dormant, `"extra": {"nested": [1, 2]}`) + ",\n" +
		"  \"google\": {\n      \"type\":   \"api_key\",\n \"key\": \"g-secret\"\n   },\n" +
		`  "xai": {"type": "api_key", "key": "x-secret"}` + "\n}\n"
	f := newMfix(t)
	f.plan = false
	f.write(f.auth, text)
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	oldNames, oldValues := rawMembers(t, text)
	newNames, newValues := rawMembers(t, f.read(f.auth))
	if !slices.Equal(newNames, oldNames) {
		t.Errorf("order %v, want %v", newNames, oldNames)
	}
	for _, n := range oldNames {
		if n == "openai" {
			continue
		}
		if newValues[n] != oldValues[n] {
			t.Errorf("%s changed:\n%s\nwant\n%s", n, newValues[n], oldValues[n])
		}
	}
	wantOpenAI := fieldsOf(t, oldValues["openai"])
	delete(wantOpenAI, "key")
	if got := fieldsOf(t, newValues["openai"]); !reflect.DeepEqual(got, wantOpenAI) {
		t.Errorf("openai's fields %v, want %v", got, wantOpenAI)
	}
}

// Step 3 writes the top level as gobble does, LF, two spaces and a final
// newline, keeps an unchanged one-line member's bytes, and lays the edited
// entry out as gobble's writes do (rows M10, M61).
func TestMigrateLayout(t *testing.T) {
	text := "{\r\n    \"xai\": {\"type\": \"api_key\", \"key\": \"x\"},\r\n    \"openai\": {\r\n" +
		"        \"type\": \"oauth\",\r\n        \"access\": \"a\",\r\n        \"key\": \"k-secret\"\r\n    }\r\n}"
	want := "{\n  \"xai\": {\"type\": \"api_key\", \"key\": \"x\"},\n  \"openai\": {\n    \"type\": \"oauth\",\n" +
		"    \"access\": \"a\"\n  }\n}\n"
	f := newMfix(t)
	f.plan = false
	f.write(f.auth, text)
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if got := f.read(f.auth); got != want {
		t.Errorf("auth.json =\n%q\nwant\n%q", got, want)
	}
}

// The backup holds the old bytes, owner-only, named in UTC whatever the
// clock's zone, and is made before auth.json is written: a write that fails
// after it leaves auth.json unchanged and the backup in place. Each failing
// write backs up again, one backup per attempt (rows M34, M35, M38, M59).
func TestMigrateBacksUpFirst(t *testing.T) {
	text := authFile(entryText("anthropic", `"type": "oauth"`, `"access": "a-secret"`, dormant))
	f := newMfix(t)
	f.write(f.auth, text)
	f.now = time.Date(2026, 11, 22, 1, 2, 3, 0, time.FixedZone("UTC-5", -5*3600))
	setHooks(t, func() { migrateHooks.failAuthWrite = func() error { return errFault } })
	if _, err := f.run(); !errors.Is(err, errFault) {
		t.Fatalf("err = %v, want the injected fault", err)
	}
	if f.read(f.auth) != text {
		t.Error("auth.json changed although its write failed")
	}
	name := filepath.Join(f.config, "auth.json.bak-20261122T060203Z")
	if got := f.backups(); !slices.Equal(got, []string{name}) {
		t.Fatalf("backups %v, want %s, named in UTC", got, name)
	}
	if f.read(name) != text {
		t.Error("the backup does not hold the old bytes")
	}
	if ok, err := appdirs.FileIsOwnerOnly(name); err != nil || !ok {
		t.Errorf("backup owner-only %v, %v; want the user's alone", ok, err)
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(name); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("backup mode %v, %v; want 0600", fi, err)
		}
	}

	f.now = f.now.Add(time.Second)
	if _, err := f.run(); !errors.Is(err, errFault) {
		t.Fatalf("second run: err = %v, want the injected fault", err)
	}
	migrateHooks.failAuthWrite = nil
	f.now = f.now.Add(time.Second)
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	backups := f.backups()
	if len(backups) != 3 {
		t.Fatalf("backups %v, want three, one per attempt", backups)
	}
	for _, b := range backups {
		if f.read(b) != text {
			t.Errorf("%s does not hold the old bytes", b)
		}
	}
	if strings.Contains(f.read(f.auth), "sk-dormant") {
		t.Error("the key is still in auth.json")
	}
}

// A second run after a completed migration writes nothing, makes no backup,
// logs nothing and carries nothing (row M11).
func TestMigrateIsIdempotent(t *testing.T) {
	f := newMfix(t)
	f.settings = [2]bool{true, true}
	f.write(f.auth, authFile(planMember(dormant)))
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	snapshot := map[string]string{}
	root := filepath.Dir(f.config)
	for _, n := range f.names() {
		snapshot[n] = f.read(filepath.Join(root, n))
	}
	f.log = syncBuf{}
	f.now = f.now.Add(time.Minute)
	r, err := f.run()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r, Report{}) {
		t.Errorf("second Report %+v, want nothing done", r)
	}
	again := map[string]string{}
	for _, n := range f.names() {
		again[n] = f.read(filepath.Join(root, n))
	}
	if !reflect.DeepEqual(again, snapshot) {
		t.Errorf("the second run changed the files:\n%v\nwant\n%v", slices.Sorted(maps.Keys(again)), slices.Sorted(maps.Keys(snapshot)))
	}
	if f.carries.Load() != 1 || f.log.String() != "" {
		t.Errorf("carries %d, log %q; want the first run's one carry and no record", f.carries.Load(), f.log.String())
	}
}

// clientsPath and hostIDPath are the state files' paths.
func (f *mfix) clientsPath() string { return filepath.Join(f.state, "clients.json") }
func (f *mfix) hostIDPath() string  { return filepath.Join(f.state, "host-id") }

// clients are clients.json's records.
func (f *mfix) clients() map[string]ClientRecord {
	f.t.Helper()
	m := map[string]ClientRecord{}
	if err := jsonx.UnmarshalStrict([]byte(f.read(f.clientsPath())), &m); err != nil {
		f.t.Fatal(err)
	}
	return m
}

// The plan entry's ids are adopted as the owner answered Q3 (rows M14 to
// M21, M23, M57).
func TestMigrateAdoptsIDs(t *testing.T) {
	t.Run("both ids", adoptIDsBothIDs)
	t.Run("host-id the same", adoptIDsHostIDTheSame)
	t.Run("host-id another", adoptIDsHostIDAnother)
	t.Run("record exists", adoptIDsRecordExists)
	t.Run("other records", adoptIDsOtherRecords)
	t.Run("hostId alone", adoptIDsHostIDAlone)
	t.Run("clientId alone", adoptIDsClientIDAlone)
	t.Run("host-id not an id", adoptIDsHostIDNotAnID)
	t.Run("entry's hostId not an id", adoptIDsEntryHostIDNotAnID)
}

// adoptIDsBothIDs is TestMigrateAdoptsIDs's case 'both ids' (M14, M23).
func adoptIDsBothIDs(t *testing.T) {
	f := newMfix(t)
	text := authFile(planMember())
	f.write(f.auth, text)
	r, err := f.run()
	if err != nil {
		t.Fatal(err)
	}
	if got := f.read(f.hostIDPath()); got != hostA+"\n" {
		t.Errorf("host-id %q, want %q", got, hostA)
	}
	if ok, err := appdirs.FileIsOwnerOnly(f.hostIDPath()); err != nil || !ok {
		t.Errorf("host-id owner-only %v, %v", ok, err)
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(f.hostIDPath()); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("host-id mode %v, %v; want 0600", fi, err)
		}
	}
	if got := f.clients(); !reflect.DeepEqual(got, map[string]ClientRecord{"openai": {ClientID: clientA, HostID: hostA}}) {
		t.Errorf("clients.json %v", got)
	}
	if f.read(f.auth) != text {
		t.Error("auth.json changed; the entry keeps both ids")
	}
	if len(f.backups()) != 0 {
		t.Error("a backup was made; only step 4 had work")
	}
	if !r.HostID || !r.ClientRecord || r.Consent || r.Backup != "" || len(r.Dropped) != 0 {
		t.Errorf("Report %+v", r)
	}
}

// adoptIDsHostIDTheSame is TestMigrateAdoptsIDs's case 'host-id the same' (M17).
func adoptIDsHostIDTheSame(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(planMember()))
	f.write(f.hostIDPath(), hostA+"\n")
	r, err := f.run()
	if err != nil {
		t.Fatal(err)
	}
	if r.HostID || !r.ClientRecord {
		t.Errorf("Report %+v, want the record alone written", r)
	}
	if got := f.clients(); got["openai"] != (ClientRecord{ClientID: clientA, HostID: hostA}) {
		t.Errorf("clients.json %v", got)
	}
}

// adoptIDsHostIDAnother is TestMigrateAdoptsIDs's case 'host-id another' (M18).
func adoptIDsHostIDAnother(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(planMember()))
	f.write(f.hostIDPath(), hostB+"\n")
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if got := f.read(f.hostIDPath()); got != hostB+"\n" {
		t.Errorf("host-id %q, want %q kept", got, hostB)
	}
	missing(t, f.clientsPath())
}

// adoptIDsRecordExists is TestMigrateAdoptsIDs's case 'record exists' (M20).
func adoptIDsRecordExists(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(planMember()))
	old := "{\n  \"openai\": {\n    \"clientId\": \"app_other\",\n    \"hostId\": \"" + hostA + "\"\n  }\n}\n"
	f.write(f.clientsPath(), old)
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if f.read(f.clientsPath()) != old {
		t.Error("clients.json changed")
	}
	if got := f.read(f.hostIDPath()); got != hostA+"\n" {
		t.Errorf("host-id %q", got)
	}
}

// adoptIDsOtherRecords is TestMigrateAdoptsIDs's case 'other records' (M21).
func adoptIDsOtherRecords(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(planMember()))
	f.write(f.clientsPath(), "{\"grok\": {\"clientId\": \"g\", \"hostId\": \""+hostB+"\"}}\n")
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	want := map[string]ClientRecord{"grok": {ClientID: "g", HostID: hostB}, "openai": {ClientID: clientA, HostID: hostA}}
	if got := f.clients(); !reflect.DeepEqual(got, want) {
		t.Errorf("clients.json %v, want %v", got, want)
	}
}

// adoptIDsHostIDAlone is TestMigrateAdoptsIDs's case 'hostId alone' (M15).
func adoptIDsHostIDAlone(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(entryText("openai", without(planFields(), "clientId")...)))
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if got := f.read(f.hostIDPath()); got != hostA+"\n" {
		t.Errorf("host-id %q", got)
	}
	missing(t, f.clientsPath())
}

// adoptIDsClientIDAlone is TestMigrateAdoptsIDs's case 'clientId alone' (M16).
func adoptIDsClientIDAlone(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(entryText("openai", without(planFields(), "hostId")...)))
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if got := f.stateFiles(); len(got) != 0 {
		t.Errorf("state files %v, want none", got)
	}
}

// adoptIDsHostIDNotAnID is TestMigrateAdoptsIDs's case 'host-id not an id' (M19).
func adoptIDsHostIDNotAnID(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(planMember()))
	f.write(f.hostIDPath(), "not an id\n")
	if _, err := f.run(); err == nil {
		t.Error("no error")
	}
	if got := f.read(f.hostIDPath()); got != "not an id\n" {
		t.Errorf("host-id replaced: %q", got)
	}
	missing(t, f.clientsPath())
}

// adoptIDsEntryHostIDNotAnID is TestMigrateAdoptsIDs's case "entry's hostId not an id" (M57).
func adoptIDsEntryHostIDNotAnID(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(entryText("openai", append(without(planFields(), "hostId"), `"hostId": "host-1234"`)...)))
	if _, err := f.run(); err == nil {
		t.Error("no error")
	}
	if got := f.stateFiles(); len(got) != 0 {
		t.Errorf("state files %v, want none", got)
	}
}

func missing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("%s exists (%v)", path, err)
	}
}

// Another application's ChatGPT login under openai has its dormant key
// dropped, and nothing adopted or carried from it (row M13).
func TestMigrateSkipsForeignEntry(t *testing.T) {
	f := newMfix(t)
	f.plan = false
	f.settings = [2]bool{true, true}
	f.write(f.auth, authFile(entryText("openai", `"type": "oauth"`, `"access": "c-secret"`, `"kind": "chatgpt"`,
		`"clientId": "app_codex"`, `"hostId": "`+hostA+`"`, dormant)))
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(f.read(f.auth), "sk-dormant") {
		t.Error("the key was not dropped")
	}
	if got := f.stateFiles(); len(got) != 0 || f.carries.Load() != 0 {
		t.Errorf("state %v, carries %d; want none from a foreign entry", got, f.carries.Load())
	}
}

// A clients.json that does not parse is an error, and is not replaced
// (row M22).
func TestMigrateBadClientsFileIsAnError(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(planMember()))
	path := filepath.Join(f.state, "clients.json")
	f.write(path, "not json\n")
	if _, err := f.run(); err == nil {
		t.Error("no error")
	}
	if f.read(path) != "not json\n" {
		t.Error("clients.json replaced")
	}
}

// readyToCarry is a plan entry whose ids are already adopted, with the
// setting true: a carry alone.
func readyToCarry(t *testing.T, extra ...string) *mfix {
	t.Helper()
	f := newMfix(t)
	f.settings = [2]bool{true, true}
	f.write(f.auth, authFile(planMember(extra...)))
	f.write(filepath.Join(f.state, "host-id"), hostA+"\n")
	f.write(filepath.Join(f.state, "clients.json"), "{\"openai\": {\"clientId\": \""+clientA+"\", \"hostId\": \""+hostA+"\"}}\n")
	return f
}

// The consent is carried as the owner answered Q4 and Q6 (rows M24 to M27,
// M32, M33, M59, M62, M64).
func TestMigrateCarriesConsent(t *testing.T) {
	t.Run("carried", carriesCarried)
	for _, answer := range []string{"declined", "granted"} { // M25
		t.Run("answered "+answer, func(t *testing.T) {
			f := readyToCarry(t, `"consents": "test.consent=`+answer+`"`)
			before := f.read(f.auth)
			if _, err := f.run(); err != nil {
				t.Fatal(err)
			}
			if f.carries.Load() != 0 || f.read(f.auth) != before {
				t.Errorf("carries %d; auth.json changed %v; want neither", f.carries.Load(), f.read(f.auth) != before)
			}
		})
	}
	for name, s := range map[string][2]bool{"setting false": {false, true}, "setting absent": {false, true}} { // M26
		t.Run(name, func(t *testing.T) {
			f := readyToCarry(t)
			f.settings = s
			if _, err := f.run(); err != nil {
				t.Fatal(err)
			}
			if f.carries.Load() != 0 {
				t.Error("CarryConsent was called")
			}
		})
	}
	for name, text := range map[string]string{ // M27
		"no openai entry": authFile(entryText("anthropic", `"type": "api_key"`, `"key": "a-secret"`)),
		"api_key entry":   authFile(entryText("openai", `"type": "api_key"`, `"key": "o-secret"`)),
		"foreign session": authFile(entryText("openai", `"type": "oauth"`, `"access": "c-secret"`, `"kind": "chatgpt"`)),
	} {
		t.Run(name, func(t *testing.T) {
			f := newMfix(t)
			f.settings = [2]bool{true, true}
			f.plan = name != "foreign session"
			f.write(f.auth, text)
			if _, err := f.run(); err != nil {
				t.Fatal(err)
			}
			if f.carries.Load() != 0 {
				t.Error("CarryConsent was called")
			}
		})
	}
	t.Run("a kind member", carriesAKindMember)
	t.Run("fails twice", carriesFailsTwice)
	t.Run("fails after a dropped key", carriesFailsAfterADroppedKey)
}

// carriesCarried is TestMigrateCarriesConsent's case 'carried' (M24, M33, M62, M64).
func carriesCarried(t *testing.T) {
	f := readyToCarry(t)
	r, err := f.run()
	if err != nil {
		t.Fatal(err)
	}
	if f.carries.Load() != 1 || !r.Consent {
		t.Errorf("carries %d, Report %+v; want one carry", f.carries.Load(), r)
	}
	if f.lockFreeAtCarry.Load() != 1 {
		t.Error("auth.json.lock was held during CarryConsent")
	}
	if e, _ := mustLoad(t, f.auth).Entry("openai"); e.Consents == "" {
		t.Error("the session holds no answer after the carry")
	}
	if len(f.backups()) != 0 {
		t.Error("a backup was made for a carry alone")
	}
}

// carriesAKindMember is TestMigrateCarriesConsent's case 'a kind member' (M62).
func carriesAKindMember(t *testing.T) {
	f := readyToCarry(t, `"kind": "chatgpt_plan"`)
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if f.carries.Load() != 0 {
		t.Error("carried into an entry that has a kind member")
	}
}

// carriesFailsTwice is TestMigrateCarriesConsent's case 'fails twice' (M32, M59).
func carriesFailsTwice(t *testing.T) {
	f := readyToCarry(t)
	f.carryErr = errFault
	for i := range 2 {
		f.now = f.now.Add(time.Second)
		if _, err := f.run(); !errors.Is(err, errFault) {
			t.Fatalf("run %d: err = %v, want CarryConsent's", i, err)
		}
	}
	if f.carries.Load() != 2 || len(f.backups()) != 0 {
		t.Errorf("carries %d, backups %v; want two carries and no backup", f.carries.Load(), f.backups())
	}
}

// carriesFailsAfterADroppedKey is TestMigrateCarriesConsent's case 'fails after a dropped key' (M32).
func carriesFailsAfterADroppedKey(t *testing.T) {
	f := readyToCarry(t, dormant)
	f.carryErr = errFault
	if _, err := f.run(); !errors.Is(err, errFault) {
		t.Fatalf("err = %v, want CarryConsent's", err)
	}
	if strings.Contains(f.read(f.auth), "sk-dormant") {
		t.Error("the key's removal did not stand")
	}
	f.carryErr = nil
	f.now = f.now.Add(time.Second)
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if f.carries.Load() != 2 || len(f.backups()) != 1 {
		t.Errorf("carries %d, backups %v; want the carry again and the first run's backup alone", f.carries.Load(), f.backups())
	}
}

func mustLoad(t *testing.T, path string) *File {
	t.Helper()
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// A settings.yaml that cannot be read stops a migration that has a consent
// to carry, with nothing written; with no unanswered plan session it is not
// asked, and the migration goes on (rows M28, M29).
func TestMigrateUnreadableSettingsWritesNothing(t *testing.T) {
	f := newMfix(t)
	f.settings = [2]bool{false, false}
	text := authFile(planMember(dormant))
	f.write(f.auth, text)
	before := f.names()
	if _, err := f.run(); err == nil {
		t.Error("no error")
	}
	if f.read(f.auth) != text || !slices.Equal(f.names(), before) {
		t.Errorf("files %v, auth.json changed %v; want nothing written", f.names(), f.read(f.auth) != text)
	}

	g := newMfix(t)
	g.settings = [2]bool{false, false}
	g.write(g.auth, authFile(planMember(dormant, `"consents": "test.consent=granted"`)))
	if _, err := g.run(); err != nil {
		t.Fatal(err)
	}
	if g.settingsCalls.Load() != 0 {
		t.Error("Settings was asked with no consent to carry")
	}
	if strings.Contains(g.read(g.auth), "sk-dormant") {
		t.Error("the key was not dropped")
	}
}

// A file at the backup's name is an error, checked before the rename, and
// nothing is written; that file is unchanged and no temporary file is left
// (row M36).
func TestMigrateBackupNameTaken(t *testing.T) {
	f := newMfix(t)
	text := authFile(entryText("anthropic", `"type": "oauth"`, dormant))
	f.write(f.auth, text)
	taken := filepath.Join(f.config, "auth.json.bak-20261122T090807Z")
	f.write(taken, "other\n")
	if _, err := f.run(); err == nil {
		t.Error("no error")
	}
	if f.read(f.auth) != text || f.read(taken) != "other\n" {
		t.Error("auth.json or the existing file changed")
	}
	noneOf(t, "a taken name", f.names(), ".tmp")
}

// A backup that cannot be written stops the migration: neither a backup nor
// its temporary file remains, auth.json is unchanged, no state file, no
// carry (row M37).
func TestMigrateBackupFailureWritesNothing(t *testing.T) {
	f := newMfix(t)
	f.settings = [2]bool{true, true}
	text := authFile(planMember(dormant))
	f.write(f.auth, text)
	setHooks(t, func() { migrateHooks.failBackup = func() error { return errFault } })
	if _, err := f.run(); !errors.Is(err, errFault) {
		t.Fatalf("err = %v, want the injected fault", err)
	}
	if f.read(f.auth) != text {
		t.Error("auth.json changed")
	}
	noneOf(t, "a failed backup", f.names(), ".bak-", ".tmp")
	if got := f.stateFiles(); len(got) != 0 || f.carries.Load() != 0 {
		t.Errorf("state %v, carries %d; want none", got, f.carries.Load())
	}
}

// A state file that cannot be written is an error; auth.json's changes and
// the backup stand, and the next run that can write the state directory
// writes the state files with no new backup (row M39).
func TestMigrateStateWriteFailureIsAnError(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(planMember(dormant)))
	f.write(f.state, "a file, not a directory\n")
	if _, err := f.run(); err == nil {
		t.Fatal("no error")
	}
	if strings.Contains(f.read(f.auth), "sk-dormant") || len(f.backups()) != 1 {
		t.Fatalf("auth.json migrated %v, backups %v; want both", !strings.Contains(f.read(f.auth), "sk-dormant"), f.backups())
	}
	if err := os.Remove(f.state); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Second)
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if got := f.read(filepath.Join(f.state, "host-id")); got != hostA+"\n" || len(f.backups()) != 1 {
		t.Errorf("host-id %q, backups %v; want the id and no new backup", got, f.backups())
	}
}

// A run stopped after auth.json's write and before step 4 leaves the rest to
// the next run, which adopts the ids with no backup (row M40).
func TestMigrateCompletesAfterInterruption(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(planMember(dormant)))
	setHooks(t, func() { migrateHooks.afterAuthWrite = func() error { return errFault } })
	if _, err := f.run(); !errors.Is(err, errFault) {
		t.Fatalf("err = %v, want the stop", err)
	}
	if got := f.stateFiles(); len(got) != 0 {
		t.Fatalf("state %v, want none before step 4", got)
	}
	migrateHooks.afterAuthWrite = nil
	f.now = f.now.Add(time.Second)
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if got := f.read(filepath.Join(f.state, "host-id")); got != hostA+"\n" || len(f.backups()) != 1 {
		t.Errorf("host-id %q, backups %v; want the id adopted and the first run's backup alone", got, f.backups())
	}
	missingNot(t, filepath.Join(f.state, "clients.json"))
}

func missingNot(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("%s: %v; want it written", path, err)
	}
}

// A cancelled run stops before its next step, a write already begun
// completing; the next run does what remains (row M51).
func TestMigrateStopsWhenCancelled(t *testing.T) {
	f := newMfix(t)
	text := authFile(planMember(dormant))
	f.write(f.auth, text)
	before := f.names()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Migrate(ctx, f.opts()); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the context's", err)
	}
	if f.read(f.auth) != text || !slices.Equal(f.names(), before) {
		t.Fatal("a cancelled run wrote")
	}

	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	setHooks(t, func() { migrateHooks.afterAuthWrite = func() error { cancel(); return nil } })
	if _, err := Migrate(ctx, f.opts()); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the context's", err)
	}
	if strings.Contains(f.read(f.auth), "sk-dormant") || len(f.backups()) != 1 {
		t.Fatal("the write begun before the cancellation did not complete")
	}
	if got := f.stateFiles(); len(got) != 0 {
		t.Fatalf("state %v, want none after the cancellation", got)
	}
	migrateHooks.afterAuthWrite = nil
	f.now = f.now.Add(time.Second)
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if got := f.read(filepath.Join(f.state, "host-id")); got != hostA+"\n" || len(f.backups()) != 1 {
		t.Errorf("host-id %q, backups %v; want the id and no new backup", got, f.backups())
	}
}

// Two runs over one file write it once: the second, started between the
// first's read and its lock, migrates the file, and the first re-reads it
// under the lock and finds nothing to do (row M42). Ten rounds, for -race.
func TestMigrateConcurrentRunsWriteOnce(t *testing.T) {
	for round := range 10 {
		f := newMfix(t)
		f.noLog = true
		f.write(f.auth, authFile(planMember(dormant)))
		var writes atomic.Int32
		var started atomic.Bool
		var secondErr error
		second := f.opts()
		second.Now = func() time.Time { return f.now.Add(time.Second) }
		setHooks(t, func() {
			migrateHooks.afterAuthWrite = func() error { writes.Add(1); return nil }
			migrateHooks.afterFirstRead = func() {
				if !started.CompareAndSwap(false, true) {
					return
				}
				done := make(chan struct{})
				go func() {
					defer close(done)
					_, secondErr = Migrate(t.Context(), second)
				}()
				<-done
			}
		})
		_, err := f.run()
		if err != nil || secondErr != nil {
			t.Fatalf("round %d: errors %v, %v", round, err, secondErr)
		}
		if n := writes.Load(); n != 1 || len(f.backups()) != 1 {
			t.Fatalf("round %d: %d writes, backups %v; want one of each", round, n, f.backups())
		}
		if strings.Contains(f.read(f.auth), "sk-dormant") {
			t.Fatalf("round %d: the file is not migrated", round)
		}
	}
}

// The migration racing HostID keeps the first id: HostID, run between the
// migration's check and its create, makes host-id, which the migration does
// not replace, and no client record names another host id (rows M43, M18).
func TestMigrateHostIDRaceKeepsFirst(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(planMember()))
	var first string
	var firstErr error
	setHooks(t, func() {
		migrateHooks.beforeHostIDCreate = func() {
			done := make(chan struct{})
			go func() {
				defer close(done)
				first, firstErr = HostID(f.state)
			}()
			<-done
		}
	})
	r, err := f.run()
	if err != nil || firstErr != nil {
		t.Fatalf("errors %v, %v", err, firstErr)
	}
	if first == "" || first == hostA {
		t.Fatalf("HostID = %q; the hook did not run first", first)
	}
	if got := f.read(filepath.Join(f.state, "host-id")); got != first+"\n" {
		t.Errorf("host-id %q, want HostID's %q", got, first)
	}
	missing(t, filepath.Join(f.state, "clients.json"))
	if r.HostID || r.ClientRecord {
		t.Errorf("Report %+v, want nothing adopted", r)
	}
}

// A client record another process saves between step 4's check and its save
// is kept: the save is made only when, under clients.json.lock, the file
// still has no openai record (row M67).
func TestMigrateClientRaceKeepsFirst(t *testing.T) {
	f := newMfix(t)
	f.write(f.auth, authFile(planMember()))
	clients := Clients{Path: filepath.Join(f.state, "clients.json")}
	theirs := ClientRecord{ClientID: "app_other", HostID: hostA}
	var saveErr error
	fired := false
	setHooks(t, func() {
		migrateHooks.beforeClientSave = func() {
			fired = true
			done := make(chan struct{})
			go func() {
				defer close(done)
				saveErr = clients.Save(t.Context(), "openai", theirs)
			}()
			<-done
		}
	})
	r, err := f.run()
	if err != nil || saveErr != nil {
		t.Fatalf("errors %v, %v", err, saveErr)
	}
	if !fired {
		t.Fatal("the other process's save never ran: the hook did not fire")
	}
	if got, err := clients.Load(t.Context(), "openai"); err != nil || got != theirs {
		t.Errorf("openai's record %+v, %v; want the other process's %+v kept", got, err, theirs)
	}
	if r.ClientRecord {
		t.Errorf("Report %+v, want no record written", r)
	}
}

// A symbolic link at auth.json is resolved: the target is migrated and
// backed up beside it, the link stays a link, and nothing is beside the link
// but auth.json.lock (row M63).
func TestMigrateFollowsSymlink(t *testing.T) {
	f := newMfix(t)
	f.plan = false
	target := filepath.Join(filepath.Dir(f.config), "real", "creds.json")
	f.write(target, authFile(entryText("anthropic", `"type": "oauth"`, dormant)))
	if err := os.MkdirAll(f.config, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, f.auth); err != nil {
		t.Skipf("no symbolic link here: %v", err)
	}
	if _, err := f.run(); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(f.auth); err != nil || fi.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("auth.json is no longer a link: %v, %v", fi, err)
	}
	if got, _ := os.Readlink(f.auth); got != target {
		t.Errorf("the link points at %q, want %q", got, target)
	}
	if strings.Contains(f.read(target), "sk-dormant") {
		t.Error("the target keeps the key")
	}
	if m, _ := filepath.Glob(filepath.Join(filepath.Dir(target), "auth.json.bak-*")); len(m) != 1 {
		t.Errorf("backups beside the target %v, want one", m)
	}
	des, err := os.ReadDir(f.config)
	if err != nil {
		t.Fatal(err)
	}
	var beside []string
	for _, de := range des {
		beside = append(beside, de.Name())
	}
	if !slices.Equal(beside, []string{"auth.json", "auth.json.lock"}) {
		t.Errorf("beside the link: %v, want auth.json and auth.json.lock", beside)
	}
}

// migrate.go reaches the SDK only through its hooks: it imports no
// llm/provider and calls no internal/auth name whose body does,
// AcceptsLogin and the methods of Logins and ClientLogins (0052-MADR D9
// step 5, Q2). Clients' methods use no llm/provider.
func TestMigrateCallsOnlyItsHooks(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "migrate.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range file.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); strings.HasSuffix(p, "/llm/provider") || strings.Contains(p, "go-llmprovider-sdk") {
			t.Errorf("migrate.go imports %s", p)
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && slices.Contains([]string{"AcceptsLogin", "Logins", "ClientLogins"}, id.Name) {
			t.Errorf("%s: migrate.go uses %s", fset.Position(id.Pos()), id.Name)
		}
		return true
	})
}
