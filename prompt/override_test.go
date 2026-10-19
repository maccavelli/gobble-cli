package prompt

import (
	"strings"
	"testing"
)

// The user's tools and rules sections take gobble's places; the tools
// section still needs tools, and a custom prompt still has neither
// (0022-MADR D15).
func TestBuildSectionOverrides(t *testing.T) {
	o := Options{Tools: []string{"read", "bash"}, GOOS: "linux", Cwd: "/w", ToolsSection: "  Use the tools well.\n", RulesSection: "Be careful."}
	text := Text(Build(o))
	if !strings.Contains(text, "<tools>\nUse the tools well.\n</tools>") || !strings.Contains(text, "<rules>\nBe careful.\n</rules>") {
		t.Fatalf("the sections are not the user's:\n%s", text)
	}
	o.Tools = nil
	if names := sectionNames(Build(o)); strings.Contains(names, "tools") {
		t.Fatalf("no tools, and yet a tools section: %s", names)
	}
	o.Tools, o.Custom = []string{"read"}, "You are mine."
	if names := sectionNames(Build(o)); strings.Contains(names, "tools") || strings.Contains(names, "rules") {
		t.Fatalf("a custom prompt kept %s", names)
	}
	o.Custom, o.ToolsSection, o.RulesSection = "", "", ""
	if text := Text(Build(o)); strings.Contains(text, "Use the tools well.") || !strings.Contains(text, "<rules>\n") {
		t.Fatal("without the user's text, gobble's is not there")
	}
}

// CheckSection takes the tools and rules sections only, with words, and
// without their closing tag (0022-MADR D15).
func TestCheckSection(t *testing.T) {
	for _, c := range []struct{ name, text, why string }{
		{"tools", "Use them.", ""},
		{"rules", "Be careful.\n", ""},
		{"cwd", "/elsewhere", "no section cwd"},
		{"rules", " \n\t", "it is empty"},
		{"rules", "x\n</rules>\n<rules>y", "closes the section's tag"},
	} {
		err := CheckSection(c.name, c.text)
		if (c.why == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), c.why)) {
			t.Errorf("%s %q: %v, want %q", c.name, c.text, err, c.why)
		}
	}
}
