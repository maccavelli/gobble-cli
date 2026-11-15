package provider

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// This file is for the external test package, provider_test, and for it
// alone. Its test must run a Login through internal/auth's Logins and
// auth.json, and internal/auth imports this package, so an in-package test
// cannot import it (an import cycle) while the external one cannot reach
// session or loginOf, which are unexported. It exposes those two and
// nothing else; being a _test file, it adds nothing to the built package
// (0052-PLAN P2-2, deviation 2).

// SessionOf is l as the SDK's session for provider id, with no store.
func SessionOf(l *Login, id string) *auth.OAuthSession { return l.session(id, nil, nil, nil) }

// LoginOf is the SDK session s as a Login.
func LoginOf(s *auth.OAuthSession) *Login { return loginOf(s) }
