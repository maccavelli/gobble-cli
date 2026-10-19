package cli

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// titleOptions are the small model that names a new session from its
// first prompt (0005-PLAN F5-5): smallModel's, its provider built as the
// session's is, by its own id. None without smallModel, and none, with a
// warning, when its provider has no credential (decision 10).
func (e *runEnv) titleOptions() acpserver.TitleOptions {
	small := e.processSettings("").Settings.SmallModel
	if small == "" {
		return acpserver.TitleOptions{}
	}
	id, model, _ := strings.Cut(small, "/") // the setting's rule has checked its shape
	if !e.hasCredential(id) {
		e.log().Warn("settings", slog.String("message", fmt.Sprintf("smallModel %s: %s has no credential, so no session is named", small, id)))
		return acpserver.TitleOptions{}
	}
	return acpserver.TitleOptions{Provider: smallProvider(e, id, model), ProviderID: id, Model: model}
}

// smallProvider builds the small model's provider for a session, as
// modelProvider builds the session's: the provider's credential or login,
// models.json's base URL and headers, the session's id and retry settings.
// Tests replace it.
var smallProvider = func(e *runEnv, id, model string) func(sessionID string, retry llm.Retry) (llm.Provider, error) {
	return func(sessionID string, retry llm.Retry) (llm.Provider, error) {
		key, _ := e.credentialKey(id) //nolint:errcheck // titleOptions found a credential; a login has no key
		o := provider.Options{ClientName: appName, ClientVersion: identity().Version, Logger: e.log(), SessionID: sessionID,
			Logins: e.logins(), Retry: &retry, APIKey: key, Model: model}
		o = e.providerOptions(id, o)
		if src, ok := e.credentials().Has(id); ok {
			o.CredentialSource = string(src)
		}
		return provider.New(id, o)
	}
}
