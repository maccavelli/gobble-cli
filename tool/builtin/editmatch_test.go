package builtin

import "testing"

// A second match that overlaps the first makes oldText ambiguous at every
// layer: exactly, after normalising, and escape-normalised; the file is
// left as it was (0037-MADR D8; F7). A count above two keeps its number
// (TestEditBehaviour's "duplicate").
func TestEditOverlappingDuplicate(t *testing.T) {
	const dup2 = "edit f.txt: oldText occurs 2 times; include more surrounding lines so it occurs once"
	cases := []editCase{
		{name: "exact", content: "x\n}\n}\n}\n", edits: []edit{{"}\n}\n", "Y\n"}}, wantErr: true, want: dup2},
		{name: "normalised", content: "x\n'}\n'}\n'}\n", edits: []edit{{"‘}\n‘}\n", "Y\n"}}, wantErr: true, want: dup2},
		{name: "escape-normalised", content: "x\n}\n}\n}\n", edits: []edit{{`}\n}\n`, "Y\n"}}, wantErr: true, want: dup2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { checkEdit(t, tc) })
	}
}
