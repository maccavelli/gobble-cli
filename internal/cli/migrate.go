package cli

import (
	"context"
	"errors"
	"io/fs"

	"go.yaml.in/yaml/v3"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// planBackgroundKey is the setting a build of the archived 0031-PLAN P4b-2
// work wrote for the plan consent, which the migration carries once
// (0052-MADR D9 step 5). gobble reads no other use of it: it reports the key
// as one it does not use (internal/config).
const planBackgroundKey = "chatgptPlanBackground"

// migrationProvider is the provider whose plan session the migration reads.
const migrationProvider = "openai"

// globalSetting is the migration's Settings: whether the global
// settings.yaml under configDir holds chatgptPlanBackground: true, read by
// its own parse, since the typed loader ignores a key it does not know. Only
// the YAML boolean true is true. A missing file is (false, true); a file that
// cannot be read or parsed, a duplicated key included, is (false, false). A
// project's settings.yaml is never read (0052-MADR D9's note of 2026-11-20;
// rows M30, M53, M65).
func globalSetting(configDir string) func() (bool, bool) {
	return func() (bool, bool) {
		data, err := fsx.ReadRegularFile(config.GlobalFile(configDir), fsx.ConfigLimit)
		if errors.Is(err, fs.ErrNotExist) {
			return false, true
		}
		if err != nil {
			return false, false
		}
		var doc map[string]yaml.Node
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return false, false
		}
		n, ok := doc[planBackgroundKey]
		if !ok || n.Kind != yaml.ScalarNode || n.ShortTag() != "!!bool" {
			return false, true
		}
		var v bool
		if err := n.Decode(&v); err != nil {
			return false, false
		}
		return v, true
	}
}

// planSessionHook is the migration's PlanSession over auth.json at
// authPath: llm/provider tells the plan session by the SDK's kind, inferred
// for an entry with no kind member, and gives the consent's id, whose answer
// Consents reads. No stored session gobble uses is no plan session; a store
// that cannot be read is its error (0052-MADR D9, Q2; row M12).
func planSessionHook(authPath string) func(context.Context) (bool, bool, error) {
	return func(ctx context.Context) (bool, bool, error) {
		store := auth.Logins{Path: authPath}
		st, err := provider.PlanConsent(migrationProvider, store) //nolint:contextcheck // PlanConsent takes no context; the consent's id comes from it alone (0052-PLAN P5 step 6)
		if err != nil {
			return false, false, err
		}
		if !st.Plan {
			return false, false, nil
		}
		answers, err := provider.Consents(ctx, store, migrationProvider)
		if errors.Is(err, provider.ErrNoStoredSession) {
			return false, false, nil
		}
		if err != nil {
			return false, false, err
		}
		for _, a := range answers {
			if a.ID == st.Consent {
				return true, a.Answer != provider.AnswerNotAsked, nil
			}
		}
		return true, false, nil
	}
}

// carryConsentHook is the migration's CarryConsent: the plan consent, by the
// id llm/provider gives, recorded granted on openai's stored session through
// provider.SetConsent, under the provider's refresh lock (0052-MADR D9 step
// 5; row M24).
func carryConsentHook(authPath string) func(context.Context) error {
	return func(ctx context.Context) error {
		store := auth.Logins{Path: authPath}
		st, err := provider.PlanConsent(migrationProvider, store) //nolint:contextcheck // PlanConsent takes no context; the consent's id comes from it alone (0052-PLAN P5 step 6)
		if err != nil {
			return err
		}
		return provider.SetConsent(ctx, store, migrationProvider, st.Consent, true)
	}
}
