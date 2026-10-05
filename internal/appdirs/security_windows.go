//go:build windows

package appdirs

// Provenance: magic-cli-remote internal/appdirs/security_windows.go at
// 9778cbc1 (Apache-2.0). Kept: the private DACL, its check and repair, the
// ACE-set comparison, trustee resolution, SecurePrivateFile and
// FileIsOwnerOnly. Left out: foreignTrustees and CurrentUserSID, which serve
// only magic-cli-remote's operator messages and admin socket, and the
// read-only mode of checkPrivateDir. tolerableTrustees is inlined into its
// one remaining caller. currentUserSID reports the token's Close error, and
// indexComponent ranges over an integer, for gobble's lint set. Record
// citations below are magic-cli-remote's.

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/sys/windows"
)

// privateDACL grants the owner and SYSTEM full control and nobody else, with
// inheritance severed ("P" = SDDL_PROTECTED). It is the Windows equivalent of
// mode 0700: Administrators is deliberately omitted, matching the Unix rule
// that only the owning uid may read the files (magic-cli-remote MADR 0116 D4).
const privateDACL = "D:P(A;OICI;FA;;;OW)(A;OICI;FA;;;SY)"

// privateACL parses privateDACL once per process. SetNamedSecurityInfo takes a
// parsed *ACL, not an SDDL string, so the descriptor is built here and its
// DACL extracted.
var privateACL = sync.OnceValues(func() (*windows.ACL, error) {
	sd, err := windows.SecurityDescriptorFromString(privateDACL)
	if err != nil {
		return nil, fmt.Errorf("appdirs: parse private DACL: %w", err)
	}
	acl, _, err := sd.DACL()
	if err != nil {
		return nil, fmt.Errorf("appdirs: extract private DACL: %w", err)
	}
	return acl, nil
})

// currentUserSID returns the SID of the process token's user.
func currentUserSID() (_ *windows.SID, err error) {
	// OpenCurrentProcessToken is deprecated for hiding the access it asks for;
	// TOKEN_QUERY is what it used, and all GetTokenUser needs.
	var token windows.Token
	err = windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return nil, fmt.Errorf("appdirs: open process token: %w", err)
	}
	defer func() { err = errors.Join(err, token.Close()) }()
	u, err := token.GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("appdirs: token user: %w", err)
	}
	return u.User.Sid, nil
}

// checkPrivateDir verifies dir is a real directory owned by the current user
// with the private DACL applied, repairing only the DACL. Ownership mismatch
// and reparse points are hard errors, never repaired.
func checkPrivateDir(dir string) error {
	fi, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("appdirs: %s is a symlink", dir)
	}
	// A non-symlink reparse point (junction, AF_UNIX socket, OneDrive
	// placeholder) surfaces as ModeIrregular. Refusing it is the Windows
	// analogue of the existing symlink refusal.
	if fi.Mode()&os.ModeIrregular != 0 {
		return fmt.Errorf("appdirs: %s is a reparse point", dir)
	}
	if !fi.IsDir() {
		return fmt.Errorf("appdirs: %s is not a directory", dir)
	}

	want, err := currentUserSID()
	if err != nil {
		return err
	}
	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("appdirs: read security info for %s: %w", dir, err)
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return fmt.Errorf("appdirs: read owner of %s: %w", dir, err)
	}
	if !owner.Equals(want) {
		return fmt.Errorf("appdirs: %s not owned by current user", dir)
	}

	if isPrivateDACL(sd.String(), want) {
		return nil
	}
	return applyPrivateDACL(dir)
}

// applyPrivateDACL writes the owner-only DACL to path.
//
// There is deliberately NO textual re-read afterwards. Windows resolves the
// SDDL "OW" alias to a concrete SID when the ACL is written and may add the
// AI flag, so the descriptor does not round-trip as the string it was given;
// an earlier version compared them and reported "still not private" for a
// directory it had just secured correctly (magic-cli-remote MADR 0116
// F23b/D23). SetNamedSecurityInfo returning nil is the evidence that the
// DACL was applied.
func applyPrivateDACL(path string) error {
	acl, err := privateACL()
	if err != nil {
		return err
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, acl, nil); err != nil {
		return fmt.Errorf("appdirs: set private DACL on %s: %w", path, err)
	}
	return nil
}

// isPrivateDACL reports whether sddl already grants access to nobody but the
// owner and SYSTEM, with inheritance severed.
//
// It compares the SET OF ACEs, not the descriptor string, and resolves the
// "OW" alias against the concrete owner SID — the two reasons a string
// comparison cannot work. It is best-effort by design: it decides only
// whether a redundant write can be skipped, so a false negative costs one
// extra SetNamedSecurityInfo call and can never manufacture an error.
func isPrivateDACL(sddl string, owner *windows.SID) bool {
	dacl := extractDACL(sddl)
	if dacl == "" {
		return false
	}
	flags, aces := splitDACL(dacl)
	// "P" (SDDL_PROTECTED) is what severs inheritance. Without it a parent's
	// ACEs still apply, so the directory is not private whatever else it says.
	if !strings.ContainsRune(flags, 'P') {
		return false
	}
	got := make(map[string]bool, len(aces))
	for _, ace := range aces {
		got[canonicalACE(ace, owner)] = true
	}
	for _, wantACE := range []string{
		canonicalACE("(A;OICI;FA;;;OW)", owner),
		canonicalACE("(A;OICI;FA;;;SY)", owner),
	} {
		if !got[wantACE] {
			return false
		}
	}
	// Any additional trustee means someone else has access.
	return len(got) == 2
}

// noForeignTrustee reports whether every ACE in sddl names a principal that is
// either the owner, SYSTEM, or Administrators.
//
// This is the VALIDATION counterpart to isPrivateDACL's ENFORCEMENT
// (magic-cli-remote MADR 0116 D22). What gobble creates gets the strict
// owner+SYSTEM protected DACL; a file another program wrote carries whatever
// ACL it inherited, which normally includes Administrators. An administrator
// on Windows can take ownership or read through SeBackupPrivilege regardless,
// so its presence in an ACL is not a security boundary. The boundary that
// matters — and that this enforces — is that no OTHER standard user can read
// the file.
func noForeignTrustee(sddl string, owner *windows.SID) bool {
	dacl := extractDACL(sddl)
	if dacl == "" {
		// No DACL at all means "everyone", not "nobody".
		return false
	}
	_, aces := splitDACL(dacl)
	allowed := map[string]bool{
		canonicalTrustee("OW", owner): true,
		canonicalTrustee("SY", owner): true,
		canonicalTrustee("BA", owner): true,
	}
	for _, ace := range aces {
		fields := strings.Split(strings.Trim(strings.ToUpper(strings.TrimSpace(ace)), "()"), ";")
		if len(fields) < 6 {
			return false
		}
		// Only ALLOW aces grant access; a DENY ace never widens it.
		if !strings.HasPrefix(fields[0], "A") {
			continue
		}
		if !allowed[canonicalTrustee(fields[5], owner)] {
			return false
		}
	}
	return true
}

// splitDACL separates the flag characters after "D:" from the ACE list.
func splitDACL(dacl string) (flags string, aces []string) {
	body := strings.TrimPrefix(dacl, "D:")
	i := strings.IndexByte(body, '(')
	if i < 0 {
		return body, nil
	}
	flags = body[:i]
	rest := body[i:]
	depth, start := 0, 0
	for k, r := range rest {
		switch r {
		case '(':
			if depth == 0 {
				start = k
			}
			depth++
		case ')':
			depth--
			if depth == 0 {
				aces = append(aces, rest[start:k+1])
			}
		}
	}
	return flags, aces
}

// canonicalACE normalizes an ACE so aliases and explicit SIDs compare equal.
func canonicalACE(ace string, owner *windows.SID) string {
	ace = strings.ToUpper(strings.TrimSpace(ace))
	fields := strings.Split(strings.Trim(ace, "()"), ";")
	if len(fields) < 6 {
		return ace
	}
	fields[5] = canonicalTrustee(fields[5], owner)
	return "(" + strings.Join(fields, ";") + ")"
}

// canonicalTrustee resolves an SDDL trustee to its SID string, so "SY" and
// "S-1-5-18" are the same trustee.
//
// Every alias must be resolved, not only the ones this code expects to see.
// Windows renders some account SIDs by alias when it writes SDDL: the
// built-in Administrator, RID 500, comes back as "LA" (magic-cli-remote MADR
// 0155, second amendment, measured). Left unresolved, an ACE granting that
// account its own file never equals the owner's SID, counts as a foreign
// trustee, and FileIsOwnerOnly refuses a file that is private.
//
// Resolution maps an alias to the SID it names and nothing more, so "BU" is
// still BUILTIN\Users and still foreign. OW and CO are the exception: in
// these checks they stand for the object's owner, not for the OWNER RIGHTS
// and CREATOR OWNER well-known SIDs that StringToSid would return.
func canonicalTrustee(t string, owner *windows.SID) string {
	switch t {
	case "OW", "CO":
		if owner != nil {
			return strings.ToUpper(owner.String())
		}
		return t
	case "SY":
		return "S-1-5-18"
	case "BA":
		return "S-1-5-32-544"
	}
	if strings.HasPrefix(t, "S-") {
		return t
	}
	if sid, err := windows.StringToSid(t); err == nil {
		return strings.ToUpper(sid.String())
	}
	return t
}

// SecurePrivateFile applies the owner-only private DACL to a file, the
// file-level counterpart of EnsurePrivateDir. It is idempotent: the DACL is
// written only when the current one differs.
func SecurePrivateFile(path string) error {
	path, err := absClean(path, "path")
	if err != nil {
		return err
	}
	if owner, err := currentUserSID(); err == nil {
		sd, serr := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
			windows.DACL_SECURITY_INFORMATION)
		if serr == nil && isPrivateDACL(sd.String(), owner) {
			return nil
		}
	}
	return applyPrivateDACL(path)
}

// FileIsOwnerOnly reports whether path is readable only by its owner, who must
// be the calling user.
//
// This is the Windows half of the "private to me" property that a POSIX caller
// expresses as Perm()&0o077 == 0. It exists because that mode test is
// meaningless here: files report 0666 whatever their ACL says.
func FileIsOwnerOnly(path string) (bool, error) {
	path, err := absClean(path, "path")
	if err != nil {
		return false, err
	}
	self, err := currentUserSID()
	if err != nil {
		return false, err
	}
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return false, fmt.Errorf("appdirs: read security info for %s: %w", path, err)
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return false, fmt.Errorf("appdirs: read owner of %s: %w", path, err)
	}
	if !owner.Equals(self) {
		return false, nil
	}
	return noForeignTrustee(sd.String(), self), nil
}

// extractDACL returns the "D:..." component of an SDDL string, or "" when
// absent. Components are single-letter prefixed and ordered O:G:D:S:.
func extractDACL(sddl string) string {
	i := indexComponent(sddl, "D:")
	if i < 0 {
		return ""
	}
	rest := sddl[i:]
	if j := indexComponent(rest[2:], "S:"); j >= 0 {
		return rest[:j+2]
	}
	return rest
}

// indexComponent finds an SDDL component prefix at nesting depth zero, so a
// "D:" inside an ACE string is not mistaken for the DACL marker.
func indexComponent(s, prefix string) int {
	depth := 0
	for i := range len(s) {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && i+len(prefix) <= len(s) && s[i:i+len(prefix)] == prefix {
			return i
		}
	}
	return -1
}
