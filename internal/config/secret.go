package config

import "strings"

// credentialSuffixes end a credential's key, once its last segment is
// lowercased and its "_" and "-" removed (0018-MADR D15).
var credentialSuffixes = []string{"key", "token", "secret", "password"}

// isCredentialPath reports whether path names a credential, which belongs
// in auth.json: its last segment ends in key, token, secret or password,
// ignoring case and an "_" or "-" before the suffix.
func isCredentialPath(path string) bool {
	last := path
	if _, after, ok := strings.CutLast(path, "."); ok {
		last = after
	}
	last = strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(last))
	for _, s := range credentialSuffixes {
		if strings.HasSuffix(last, s) {
			return true
		}
	}
	return false
}
