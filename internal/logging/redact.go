package logging

import (
	"log/slog"
	"strings"
)

const redacted = "[REDACTED]"

// sensitiveSuffixes are matched against a lower-case key. They cover the
// step's list (key, token, secret) and the organisation's secret-name
// suffixes, which add password.
var sensitiveSuffixes = []string{"key", "token", "secret", "password"}

// redact is the ReplaceAttr of every handler: it hides the value of a key
// that names a credential. Handlers call it for each leaf attribute, nested
// groups included, after LogValuer values are resolved.
func redact(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() != slog.KindGroup && sensitive(a.Key) {
		return slog.String(a.Key, redacted)
	}
	return a
}

func sensitive(key string) bool {
	k := strings.ToLower(key)
	if k == "authorization" {
		return true
	}
	for _, s := range sensitiveSuffixes {
		if strings.HasSuffix(k, s) {
			return true
		}
	}
	return false
}
