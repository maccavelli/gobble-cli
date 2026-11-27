package config

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// M35, M36: modelData.refreshHours and modelData.warnDays are integers, at
// least 0, global only, read at process start, defaulting to 24 and 7, in
// the template and the schema (0072-MADR D11; "Refresh 24h, warn past 7
// days (Recommended)").
func TestModelDataSettingKeys_0072(t *testing.T) {
	for _, want := range []struct {
		path string
		def  int64
		max  int64
	}{{"modelData.refreshHours", 24, 8760}, {"modelData.warnDays", 7, 3650}} {
		k, ok := KeyOf(want.path)
		if !ok || k.Kind != KindInt || k.Default != strconv.FormatInt(want.def, 10) || k.Min != 0 || k.Max != want.max ||
			k.Scope != ScopeGlobalOnly || k.Read != ReadProcess {
			t.Errorf("%s = %+v, present %v; want a global, process integer, default %d, 0 to %d", want.path, k, ok, want.def, want.max)
		}
		r, ds := Resolve(Options{})
		if v, ok := r.Settings.Value(want.path); len(ds) != 0 || !ok || v != want.def {
			t.Errorf("default %s = %v, %v (%v), want %d", want.path, v, ok, ds, want.def)
		}
	}
	if d := Defaults().ModelData; d.RefreshHours != 24 || d.WarnDays != 7 {
		t.Errorf("Defaults().ModelData = %+v", d)
	}
	if !bytes.Contains(Template(), []byte("# modelData:\n#   refreshHours: 24\n#   warnDays: 7\n")) {
		t.Error("the template has no modelData block at the defaults")
	}
	schema, err := Schema()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{`"refreshHours"`, `"warnDays"`, `"maximum": 8760`, `"maximum": 3650`} {
		if !strings.Contains(string(schema), s) {
			t.Errorf("the schema has no %s", s)
		}
	}
	// Global only: a trusted project's file cannot set them.
	l := newLayout(t, "", "modelData:\n  refreshHours: 1\n")
	r, ds := Resolve(Options{ConfigDir: l.config, Cwd: l.project, Trusted: trustAll})
	if v, _ := r.Settings.Value("modelData.refreshHours"); v != int64(24) || len(ds) == 0 {
		t.Errorf("a project set modelData.refreshHours to %v (%v); want the default and a diagnostic", v, ds)
	}
}

// M70: each bound is accepted; one past it is "must be at most N", and the
// default is used (0072-MADR D11; "Max 8760 h and 3650 days
// (Recommended)").
func TestModelDataSettingBounds_0072(t *testing.T) {
	for _, tc := range []struct {
		path  string
		value int64
		want  int64
		diag  string
	}{
		{"modelData.refreshHours", 8760, 8760, ""},
		{"modelData.refreshHours", 8761, 24, "modelData.refreshHours: must be at most 8760"},
		{"modelData.refreshHours", 0, 0, ""},
		{"modelData.warnDays", 3650, 3650, ""},
		{"modelData.warnDays", 3651, 7, "modelData.warnDays: must be at most 3650"},
		{"modelData.warnDays", 0, 0, ""},
	} {
		group, key, _ := strings.Cut(tc.path, ".")
		l := newLayout(t, fmt.Sprintf("%s:\n  %s: %d\n", group, key, tc.value), "")
		r, ds := Resolve(Options{ConfigDir: l.config, Cwd: l.project})
		v, _ := r.Settings.Value(tc.path)
		var texts []string
		for _, d := range ds {
			texts = append(texts, d.String())
		}
		got := strings.Join(texts, "\n")
		if v != tc.want || (tc.diag == "") != (got == "") || !strings.Contains(got, tc.diag) {
			t.Errorf("%s: %d: value %v, diagnostics %q; want %d and %q", tc.path, tc.value, v, got, tc.want, tc.diag)
		}
	}
}
