package render

import "testing"

func TestDisplayWidth(t *testing.T) {
	family := string(rune(0x1f468)) + string(rune(zwj)) + string(rune(0x1f469)) + string(rune(zwj)) + string(rune(0x1f467))
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"ASCII", "abc", 3},
		{"CJK", "世界", 4},
		{"fullwidth", "ＡＢ", 4},
		{"precomposed accent", "é", 1},
		{"combining accent", "e" + string(rune(0x301)), 1},
		{"emoji joined by ZWJ: each member counts, the joiners do not", family, 6},
		{"empty", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := displayWidth(tc.in); got != tc.want {
				t.Fatalf("displayWidth(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestTruncateWidth(t *testing.T) {
	cases := []struct {
		in   string
		w    int
		want string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"a longer title", 8, "a longe…"},
		{"世界世界世界", 7, "世界世…"},
		{"abc", 1, ""},
	}
	for _, tc := range cases {
		if got := truncateWidth(tc.in, tc.w, "…"); got != tc.want {
			t.Errorf("truncateWidth(%q, %d) = %q, want %q", tc.in, tc.w, got, tc.want)
		}
	}
}

func TestTablesGolden(t *testing.T) {
	in := "Before.\n\n" +
		"| Name | Lang | Count |\n" +
		"|:---|:---:|---:|\n" +
		"| gobble | Go | 1 |\n" +
		"| 世界 | 日本語 | 12345 |\n" +
		"| esc \\| pipe | x |\n" +
		"\nAfter.\n" +
		"| not | a table |\n" +
		"no delimiter row\n" +
		"| last | table |\n|---|---|\n| no | newline |"
	golden(t, "tables", Tables(in))
}

func TestTablesLeaveOtherTextAlone(t *testing.T) {
	for _, in := range []string{"", "plain\n", "| a |\n| b |\n", "|a|b|\n|-|x|\n"} {
		if got := Tables(in); got != in {
			t.Errorf("Tables(%q) = %q", in, got)
		}
	}
}
