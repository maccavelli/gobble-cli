package cli

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// opensBrowser is a sign-in as the wizard runs a browser one: it tells the
// prompter the URL, gives it to OpenURL when the caller set one (provider's
// SignIn passes the wizard no opener otherwise, signin.go's open), and saves
// the login. A caller that set no opener leaves the browser shut.
func opensBrowser(url string) signInFunc {
	return func(ctx context.Context, id string, p provider.Prompter, o provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		p.Notify("Open %s in your browser", url)
		if o.OpenURL != nil {
			if err := o.OpenURL(url); err != nil {
				return provider.Credential{}, err
			}
		}
		return provider.Credential{}, store.Save(ctx, id, &provider.Login{Access: "a-browser", Refresh: "r", Issuer: "https://auth.x.ai"})
	}
}

// showsDeviceCode is a sign-in as the wizard runs a device one: it gives the
// code, with the expiry the SDK's carries (provider's SignIn builds it from
// the wizard's DeviceCode, signin.go's show), to ShowDeviceCode, or tells the
// prompter its default words when the caller set none, and saves the login.
func showsDeviceCode(expires time.Time) signInFunc {
	return func(ctx context.Context, id string, p provider.Prompter, o provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		d := provider.DeviceCode{URL: "https://accounts.x.ai/device", Code: "ABCD-1234", Expires: expires}
		if o.ShowDeviceCode == nil {
			p.Notify("Open %s and enter the code %s", d.URL, d.Code)
		} else if err := o.ShowDeviceCode(d); err != nil {
			return provider.Credential{}, err
		}
		return provider.Credential{}, store.Save(ctx, id, &provider.Login{Access: "a-device", Refresh: "r", Issuer: "https://auth.x.ai"})
	}
}

// Every terminal surface of a browser sign-in gives its URL to the host's
// browser once, through the opener (0051-MADR D3, F56): gobble auth login,
// /login, gobble setup and the first run. The opener is the test's; no
// browser opens.
func TestLoginOpensBrowser(t *testing.T) {
	const authorize = "https://auth.x.ai/oauth2/authorize?state=s"
	var opened []string
	replace(t, &openURL, func(_ context.Context, u string) error { opened = append(opened, u); return nil })
	replace(t, &signInWith, opensBrowser(authorize))
	grok := slices.Index(provider.KeyProviders(), "grok") + 1
	once := func(t *testing.T, surface string) {
		t.Helper()
		if !slices.Equal(opened, []string{authorize}) {
			t.Errorf("%s opened %q, want the authorize URL once", surface, opened)
		}
		opened = nil
	}

	t.Run("gobble auth login grok", func(t *testing.T) {
		c := newCfgEnv(t)
		c.want(c.run("auth", "login", "grok"), ExitOK, "Logged in to grok\n", "")
		once(t, "gobble auth login grok")
	})

	t.Run("/login grok", func(t *testing.T) {
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.start(Prompt{})
		h.press("/login grok\r")
		waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "Logged in to grok") })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		once(t, "/login grok")
	})

	t.Run("gobble setup", func(t *testing.T) {
		e, _ := modelEnv(t)
		q, _, _ := asked(t, fmt.Sprintf("%d\n", grok))
		if _, err := e.signIn(t.Context(), q, ""); err != nil {
			t.Fatal(err)
		}
		once(t, "gobble setup")
	})

	t.Run("the first run", func(t *testing.T) {
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.answer = "\r"
		h.start(Prompt{})
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Sign in to which provider?") })
		h.press(fmt.Sprintf("%d\r", grok))
		waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "Logged in to grok") })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		once(t, "the first run")
	})
}

// A device code's text names when the code expires, "(expires 14:05)", as a
// time on gobble's clock: the clock's zone, not the machine's, so the
// words are the same wherever the test runs (0051-MADR D3, F56). It does so
// on gobble auth login --device and on /login; a code with no expiry names
// none.
func TestDeviceCodeTextNamesExpiry(t *testing.T) {
	expires := time.Date(2026, 10, 15, 14, 5, 0, 0, time.UTC)
	const words = "Open https://accounts.x.ai/device and enter the code ABCD-1234"
	replace(t, &signInWith, showsDeviceCode(expires))

	t.Run("gobble auth login --device", func(t *testing.T) {
		c := newCfgEnv(t)
		for _, tc := range []struct {
			name string
			zone *time.Location
			want string
		}{
			{"UTC", time.UTC, words + " (expires 14:05)\n"},
			{"+02:00", time.FixedZone("", 2*3600), words + " (expires 16:05)\n"},
		} {
			replace(t, &clock, func() time.Time { return time.Date(2026, 10, 15, 13, 50, 0, 0, tc.zone) })
			r := c.run("auth", "login", "grok", "--device")
			if r.code != ExitOK || !strings.Contains(r.stderr, tc.want) {
				t.Errorf("a clock in %s: exit %d, stderr %q, want %q", tc.name, r.code, r.stderr, tc.want)
			}
		}
	})

	t.Run("/login", func(t *testing.T) {
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.e.now = func() time.Time { return time.Date(2026, 10, 15, 13, 50, 0, 0, time.FixedZone("", -5*3600)) }
		h.start(Prompt{})
		h.press("/login grok\r")
		waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "Logged in to grok") })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		if want := words + " (expires 09:05)\n"; !strings.Contains(h.stderr.String(), want) {
			t.Errorf("/login's stderr lacks %q:\n%s", want, h.stderr)
		}
	})

	t.Run("no expiry", func(t *testing.T) {
		replace(t, &signInWith, showsDeviceCode(time.Time{}))
		c := newCfgEnv(t)
		r := c.run("auth", "login", "grok", "--device")
		if r.code != ExitOK || !strings.Contains(r.stderr, words+"\n") || strings.Contains(r.stderr, "expires") {
			t.Errorf("a code with no expiry: exit %d, stderr %q, want the words and no expiry", r.code, r.stderr)
		}
	})
}
