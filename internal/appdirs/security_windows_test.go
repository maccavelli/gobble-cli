//go:build windows

package appdirs

// Provenance: magic-cli-remote internal/appdirs/security_windows_test.go at
// 9778cbc1 (Apache-2.0). The foreignTrustees assertions go with that
// function. The LA-alias test fails, rather than skips, when the host cannot
// resolve the alias (0008-PLAN C5). Record citations are magic-cli-remote's.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// TestIsPrivateDACL: Windows resolves the "OW" alias to a concrete SID on
// write and may add the AI flag, so the round-tripped SDDL never equals the
// constant (magic-cli-remote MADR 0116 F23b). These cases pin the ACE-set
// comparison.
func TestIsPrivateDACL(t *testing.T) {
	owner, err := currentUserSID()
	if err != nil {
		t.Fatal(err)
	}
	sid := owner.String()

	for _, tc := range []struct {
		name string
		sddl string
		want bool
	}{
		{"resolved owner sid with AI flag", "O:" + sid + "G:" + sid + "D:PAI(A;OICI;FA;;;" + sid + ")(A;OICI;FA;;;SY)", true},
		{"literal constant form", "D:P(A;OICI;FA;;;" + sid + ")(A;OICI;FA;;;SY)", true},
		{"SYSTEM as an explicit sid", "D:P(A;OICI;FA;;;" + sid + ")(A;OICI;FA;;;S-1-5-18)", true},
		{"not protected", "D:AI(A;OICI;FA;;;" + sid + ")(A;OICI;FA;;;SY)", false},
		{"extra trustee has access", "D:P(A;OICI;FA;;;" + sid + ")(A;OICI;FA;;;SY)(A;OICI;FA;;;BU)", false},
		{"owner missing", "D:P(A;OICI;FA;;;SY)", false},
		{"no dacl at all", "O:" + sid + "G:" + sid, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPrivateDACL(tc.sddl, owner); got != tc.want {
				t.Errorf("isPrivateDACL(%q) = %v, want %v", tc.sddl, got, tc.want)
			}
		})
	}
}

func TestSplitDACL(t *testing.T) {
	flags, aces := splitDACL("D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	if flags != "PAI" {
		t.Errorf("flags = %q, want PAI", flags)
	}
	if len(aces) != 2 {
		t.Fatalf("aces = %v, want 2", aces)
	}
	if aces[0] != "(A;OICI;FA;;;SY)" || aces[1] != "(A;OICI;FA;;;BA)" {
		t.Errorf("aces = %v", aces)
	}
}

func TestCanonicalTrustee(t *testing.T) {
	owner, err := currentUserSID()
	if err != nil {
		t.Fatal(err)
	}
	if got := canonicalTrustee("SY", owner); got != "S-1-5-18" {
		t.Errorf("SY -> %q", got)
	}
	if got, want := canonicalTrustee("OW", owner), owner.String(); got != want {
		t.Errorf("OW -> %q, want %q", got, want)
	}
	if got := canonicalTrustee("S-1-5-18", owner); got != "S-1-5-18" {
		t.Errorf("explicit sid changed: %q", got)
	}
}

func TestSecurePrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cred.json")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SecurePrivateFile(path); err != nil {
		t.Fatalf("SecurePrivateFile: %v", err)
	}
	ok, err := FileIsOwnerOnly(path)
	if err != nil {
		t.Fatalf("FileIsOwnerOnly: %v", err)
	}
	if !ok {
		t.Error("a file just secured with the private DACL is not owner-only")
	}
	if err := SecurePrivateFile(path); err != nil {
		t.Fatalf("second SecurePrivateFile: %v", err)
	}
}

// TestNoForeignTrustee: Administrators in an inherited ACL is not a
// security boundary on Windows; another standard user is
// (magic-cli-remote MADR 0116 D22).
func TestNoForeignTrustee(t *testing.T) {
	owner, err := currentUserSID()
	if err != nil {
		t.Fatal(err)
	}
	sid := owner.String()
	other := "S-1-5-21-1111111111-2222222222-3333333333-1005"

	for _, tc := range []struct {
		name string
		sddl string
		want bool
	}{
		{"owner only", "D:P(A;;FA;;;" + sid + ")", true},
		{"owner and SYSTEM", "D:P(A;OICI;FA;;;" + sid + ")(A;OICI;FA;;;SY)", true},
		{"inherited profile ACL with Administrators", "D:AI(A;;FA;;;" + sid + ")(A;;FA;;;SY)(A;;FA;;;BA)", true},
		{"another standard user", "D:AI(A;;FA;;;" + sid + ")(A;;FA;;;" + other + ")", false},
		{"Users group", "D:AI(A;;FA;;;" + sid + ")(A;;0x1200a9;;;BU)", false},
		{"deny ace does not widen access", "D:AI(D;;FA;;;BU)(A;;FA;;;" + sid + ")", true},
		{"no dacl means everyone", "O:" + sid, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := noForeignTrustee(tc.sddl, owner); got != tc.want {
				t.Errorf("noForeignTrustee(%q) = %v, want %v", tc.sddl, got, tc.want)
			}
		})
	}
}

// A file inheriting the private DACL from its directory validates.
func TestFileIsOwnerOnlyAcceptsInheritedACL(t *testing.T) {
	home := filepath.Join(t.TempDir(), "pending")
	if err := EnsurePrivateDir(home); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "auth.json")
	if err := os.WriteFile(path, []byte(`{"k":"v"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ok, err := FileIsOwnerOnly(path)
	if err != nil {
		t.Fatalf("FileIsOwnerOnly: %v", err)
	}
	if !ok {
		t.Error("a file inheriting the private DACL did not validate as owner-only")
	}
}

// TestAliasedOwnerIsNotForeign: Windows renders the built-in Administrator,
// RID 500, as "LA" in SDDL. An ACE granting that account its own file must
// not read as a foreign trustee (magic-cli-remote MADR 0155, second
// amendment), while BUILTIN\Users, rendered BU, stays foreign.
func TestAliasedOwnerIsNotForeign(t *testing.T) {
	la, err := windows.StringToSid("LA")
	if err != nil {
		t.Fatalf("this host cannot resolve the LA alias: %v", err)
	}
	laSID := la.String()

	render := func(sddl string) string {
		t.Helper()
		sd, err := windows.SecurityDescriptorFromString(sddl)
		if err != nil {
			t.Fatalf("parse %q: %v", sddl, err)
		}
		return sd.String()
	}

	file := render("O:" + laSID + "D:P(A;;FA;;;" + laSID + ")(A;;FA;;;SY)")
	if !strings.Contains(file, ";LA)") {
		t.Fatalf("fixture did not exercise the alias; Windows rendered %q", file)
	}
	if !noForeignTrustee(file, la) {
		t.Errorf("an ACE for the owner rendered as LA was treated as foreign: %q", file)
	}

	dir := render("O:" + laSID + "D:P(A;OICI;FA;;;" + laSID + ")(A;OICI;FA;;;SY)")
	if !isPrivateDACL(dir, la) {
		t.Errorf("isPrivateDACL rejected the private DACL owned by LA: %q", dir)
	}

	exposed := render("O:" + laSID + "D:P(A;;FA;;;" + laSID + ")(A;;FR;;;BU)")
	if !strings.Contains(exposed, ";BU)") {
		t.Fatalf("fixture did not exercise the BU alias; Windows rendered %q", exposed)
	}
	if noForeignTrustee(exposed, la) {
		t.Errorf("BUILTIN\\Users became tolerated after alias resolution: %q", exposed)
	}
}
