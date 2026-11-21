package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// The first-run loop (0063-PLAN P1 step 5 and its additions A7 to A11 and
// A20; 0063-MADR D8, rows R15, R37 to R39, R42, R61 to R72). Each test runs
// over eachEnter for the questions' answers, with Gemini's sign-in through
// typedKeySignIn unless it says otherwise. Red for the loop: fails at the
// base and after P2, passes after P2b. Guards hold throughout.

// sessionPrompt is drawn on stdout when the session's prompt reads: once
// it is there, the first run has ended.
const sessionPrompt = "> "

// providerMenu is the first line of the provider menu.
const providerMenu = "Sign in to which provider?"

// firstRunHarness is a line session over a scripted agent, its first offer
// answered y and enter, Gemini's sign-in asking for a key.
func firstRunHarness(t *testing.T, enter string, turns ...acptest.Turn) (*lineHarness, *acptest.ScriptAgent) {
	t.Helper()
	var o provider.SignInOptions
	replace(t, &signInWith, typedKeySignIn(&o))
	agent := &acptest.ScriptAgent{Turns: turns}
	h := newLineHarness(t, agent)
	h.enter, h.answer = enter, "y"+enter
	return h, agent
}

// expect waits for cond while the first run goes on. It fails at once when
// the session's prompt is drawn first, and after 5 s: either way the input
// is closed, so the session ends.
func expect(t *testing.T, h *lineHarness, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		prompt := strings.Contains(h.stdout.String(), sessionPrompt)
		if prompt || time.Now().After(deadline) {
			_ = h.keys.Close()
			_ = h.wait()
			why := "within 5 s"
			if prompt {
				why = "before the session's prompt"
			}
			t.Fatalf("%s: not %s; stderr:\n%s", what, why, h.stderr)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// has reports whether the harness's stderr holds s.
func has(h *lineHarness, s string) func() bool {
	return func() bool { return strings.Contains(h.stderr.String(), s) }
}

// offeredAgain reports whether the offer was made n times, the last one
// waiting at its [y/N] hint.
func offeredAgain(h *lineHarness, n int) func() bool {
	return func() bool {
		s := h.stderr.String()
		return offerCount(h) == n && strings.HasSuffix(s, firstRunQuestion+" [y/N]: ")
	}
}

// toKeyPrompt answers the provider menu with Gemini and waits for its key
// prompt.
func toKeyPrompt(t *testing.T, h *lineHarness) {
	t.Helper()
	expect(t, h, "the provider menu", has(h, "Number [1]: "))
	h.press(strconv.Itoa(providerNumber("gemini")) + h.enterKey())
	expect(t, h, "Gemini's key prompt", has(h, "Enter your Gemini (Google) API key: "))
}

// cancelledRound runs one round cancelled by Ctrl+C at the key prompt, and
// waits for the offer again.
func cancelledRound(t *testing.T, h *lineHarness) {
	t.Helper()
	toKeyPrompt(t, h)
	h.press("\x03")
	expect(t, h, "the cancel line", has(h, "Sign-in cancelled; nothing was saved."))
	expect(t, h, "the offer again, [y/N]", offeredAgain(h, 2))
}

// toPrompt waits for the session's prompt, then ends the session with
// Ctrl+D.
func toPrompt(t *testing.T, h *lineHarness) {
	t.Helper()
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), sessionPrompt) })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
}

// wantCounts checks how many times the offer, the warning and the provider
// menu were shown.
func wantCounts(t *testing.T, h *lineHarness, offers, warns, menus int) {
	t.Helper()
	if got := offerCount(h); got != offers {
		t.Errorf("the offer was made %d times; want %d", got, offers)
	}
	if got := warningCount(h); got != warns {
		t.Errorf("the warning was printed %d times; want %d", got, warns)
	}
	if got := strings.Count(h.stderr.String(), providerMenu); got != menus {
		t.Errorf("the provider menu was shown %d times; want %d", got, menus)
	}
	if t.Failed() {
		t.Logf("stderr:\n%s", h.stderr)
	}
}

// Ctrl+C at the key prompt: the cancel line, then the offer again as
// [y/N]; n then ends it, with the warning once. Red for the loop.
func TestFirstRunOfferAgainAfterKeyCancel(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.start(Prompt{})
		cancelledRound(t, h)
		h.press("n" + enter)
		expect(t, h, "the warning", has(h, "Warning: no model credential"))
		toPrompt(t, h)
		wantCounts(t, h, 2, 1, 1)
	})
}

// Enter on an empty key prompt (Q4): the same. Red for the loop.
func TestFirstRunOfferAgainAfterEmptyKey(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.start(Prompt{})
		toKeyPrompt(t, h)
		h.press("\r")
		expect(t, h, "the cancel line", has(h, "Sign-in cancelled; nothing was saved."))
		expect(t, h, "the offer again, [y/N]", offeredAgain(h, 2))
		h.press("n" + enter)
		expect(t, h, "the warning", has(h, "Warning: no model credential"))
		toPrompt(t, h)
		wantCounts(t, h, 2, 1, 1)
	})
}

// The interrupt at the provider menu: the cancel line, then the offer
// again. Red for the loop.
func TestFirstRunOfferAgainAfterMenuInterrupt(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.start(Prompt{})
		expect(t, h, "the provider menu", has(h, "Number [1]: "))
		if !h.e.intr.take() {
			t.Fatal("the provider menu did not take the interrupt")
		}
		expect(t, h, "the cancel line", has(h, "Sign-in cancelled; nothing was saved."))
		expect(t, h, "the offer again, [y/N]", offeredAgain(h, 2))
		h.press("n" + enter)
		expect(t, h, "the warning", has(h, "Warning: no model credential"))
		toPrompt(t, h)
		wantCounts(t, h, 2, 1, 1)
	})
}

// A sign-in that fails, refused or offline: its line, then the offer
// again, with no warning between rounds. Red for the loop.
func TestFirstRunOfferAgainAfterFailure(t *testing.T) {
	for _, c := range []struct {
		name string
		err  error
		line string
	}{
		{"refused", &provider.SignInError{ID: "claude", Keys: 3}, "Error: claude refused 3 keys; none was saved"},
		{"offline", fmt.Errorf("claude: %w", llm.ErrOffline), "Error: claude: " + llm.ErrOffline.Error()},
	} {
		t.Run(c.name, func(t *testing.T) {
			eachEnter(t, func(t *testing.T, enter string) {
				h, _ := firstRunHarness(t, enter)
				replace(t, &signInWith, func(context.Context, string, provider.Prompter, provider.SignInOptions, provider.LoginStore) (provider.Credential, error) {
					return provider.Credential{}, c.err
				})
				h.start(Prompt{})
				expect(t, h, "the provider menu", has(h, "Number [1]: "))
				h.press(strconv.Itoa(providerNumber("claude")) + enter)
				expect(t, h, "the failure's line", has(h, c.line))
				expect(t, h, "the offer again, [y/N]", offeredAgain(h, 2))
				if n := warningCount(h); n != 0 {
					t.Errorf("the warning was printed between rounds")
				}
				h.press("n" + enter)
				expect(t, h, "the warning", has(h, "Warning: no model credential"))
				toPrompt(t, h)
				wantCounts(t, h, 2, 1, 1)
			})
		})
	}
}

// Q9 (b): Enter alone at the repeated offer ends the loop, with the
// warning, and shows no menu again. Red for the loop.
func TestFirstRunRepeatedOfferDefaultsNo(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.start(Prompt{})
		cancelledRound(t, h)
		h.press(enter)
		expect(t, h, "the warning", has(h, "Warning: no model credential"))
		toPrompt(t, h)
		wantCounts(t, h, 2, 1, 1)
	})
}

// n at the repeated offer: the warning once, the session's prompt, and a
// turn typed there runs. Red for the loop.
func TestFirstRunDeclineOnRepeatEnds(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter, acptest.Turn{Updates: []string{chunk("reply-1\n")}})
		h.start(Prompt{})
		cancelledRound(t, h)
		h.press("n" + enter)
		expect(t, h, "the warning", has(h, "Warning: no model credential"))
		waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), sessionPrompt) })
		h.press("hello\r")
		waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "reply-1") })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		wantCounts(t, h, 2, 1, 1)
	})
}

// Keys typed after Ctrl+C at the key prompt, in the same press, do not
// answer the repeated offer: it is shown and waits. Red for the loop.
func TestFirstRunTypeAheadDroppedBeforeRepeatedOffer(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.start(Prompt{})
		toKeyPrompt(t, h)
		h.press("\x03y\r")
		expect(t, h, "the offer again, [y/N]", offeredAgain(h, 2))
		time.Sleep(300 * time.Millisecond)
		if n := strings.Count(h.stderr.String(), providerMenu); n != 1 {
			t.Fatalf("the provider menu was shown %d times before the repeated offer was answered", n)
		}
		h.press("n" + enter)
		expect(t, h, "the warning", has(h, "Warning: no model credential"))
		toPrompt(t, h)
		wantCounts(t, h, 2, 1, 1)
	})
}

// A8 (row R38): the interrupt at a repeated offer declines it: the warning
// once, no menu again. Red for the loop.
func TestFirstRunInterruptAtRepeatedOfferEnds(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.start(Prompt{})
		cancelledRound(t, h)
		if !h.e.intr.take() {
			t.Fatal("the repeated offer did not take the interrupt")
		}
		expect(t, h, "the warning", has(h, "Warning: no model credential"))
		toPrompt(t, h)
		wantCounts(t, h, 2, 1, 1)
	})
}

// A9 (row R65): y at a repeated offer starts another round. Red for the
// loop.
func TestFirstRunYesOnRepeatStartsAnotherRound(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.start(Prompt{})
		cancelledRound(t, h)
		h.press("y" + enter)
		expect(t, h, "the provider menu again", func() bool {
			return strings.Count(h.stderr.String(), providerMenu) == 2 && strings.HasSuffix(h.stderr.String(), "Number [1]: ")
		})
		if !h.e.intr.take() {
			t.Fatal("the provider menu did not take the interrupt")
		}
		expect(t, h, "the offer a third time", offeredAgain(h, 3))
		h.press("n" + enter)
		expect(t, h, "the warning", has(h, "Warning: no model credential"))
		toPrompt(t, h)
		wantCounts(t, h, 3, 1, 2)
	})
}

// A20 (row R69): the input ends while the key prompt reads: the cancel
// line, the offer once, the warning once, and the session ends within the
// bound. Red for the loop: at the base the warning is not printed.
func TestFirstRunEndOfInputInRoundEnds(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.start(Prompt{})
		toKeyPrompt(t, h)
		if err := endSession(t, h); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(h.stderr.String(), "Sign-in cancelled; nothing was saved.") {
			t.Errorf("no cancel line")
		}
		wantCounts(t, h, 1, 1, 1)
	})
}

// Guards: the loop's ways out, which hold at the base and must hold after
// it.

// n at the first offer: the warning once, the offer once.
func TestFirstRunDeclineEnds(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.answer = "n" + enter
		h.start(Prompt{})
		toPrompt(t, h)
		wantCounts(t, h, 1, 1, 0)
	})
}

// The interrupt at the first offer: the warning once, no menu.
func TestFirstRunInterruptAtOfferEnds(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.noAnswer = true
		h.start(Prompt{})
		waitUntil(t, has(h, firstRunQuestion+" [Y/n]: "))
		if !h.e.intr.take() {
			t.Fatal("the offer did not take the interrupt")
		}
		toPrompt(t, h)
		wantCounts(t, h, 1, 1, 0)
	})
}

// The input closed after an empty key: the session ends within the bound,
// the warning printed once at most.
func TestFirstRunClosedInputEnds(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.start(Prompt{})
		toKeyPrompt(t, h)
		h.press("\r")
		if err := endSession(t, h); err != nil {
			t.Fatal(err)
		}
		if n := warningCount(h); n > 1 {
			t.Errorf("the warning was printed %d times", n)
		}
	})
}

// A sign-in that fails at once, and the input closed after y: the session
// ends within the bound.
func TestFirstRunInstantFailureClosedInputEnds(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		replace(t, &signInWith, noSignIn)
		h.start(Prompt{})
		if err := endSession(t, h); err != nil {
			t.Fatal(err)
		}
		if n := warningCount(h); n > 1 {
			t.Errorf("the warning was printed %d times", n)
		}
	})
}

// /login does not loop: an empty key cancels it, and the offer is not made
// again.
func TestLoginDoesNotLoop(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.answer = "n" + enter
		h.start(Prompt{})
		waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), sessionPrompt) })
		h.press("/login gemini\r")
		waitUntil(t, has(h, "Enter your Gemini (Google) API key: "))
		h.press("\r")
		waitUntil(t, has(h, "Sign-in cancelled; nothing was saved."))
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		if n := offerCount(h); n != 1 {
			t.Errorf("the offer was made %d times; want once", n)
		}
	})
}

// gobble setup does not loop: Ctrl+C at the key prompt ends it, and no
// sign-in is offered.
func TestSetupDoesNotLoop(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		var o provider.SignInOptions
		replace(t, &signInWith, typedKeySignIn(&o))
		e, w, stderr := pipeEnv(t)
		done := runAsync(func() error { return (&SetupCmd{}).Run(e) })
		waitUntil(t, func() bool { return strings.Contains(stderr.String(), "Number [1]: ") })
		typeInto(t, w, strconv.Itoa(providerNumber("gemini"))+enter)
		waitUntil(t, func() bool { return strings.Contains(stderr.String(), "Enter your Gemini (Google) API key: ") })
		typeInto(t, w, "\x03")
		err := waitDone(t, done, stderr)
		if err == nil || !strings.Contains(err.Error(), "setup cancelled; nothing was saved") {
			t.Fatalf("setup after Ctrl+C: %v", err)
		}
		if strings.Contains(stderr.String(), firstRunQuestion) {
			t.Fatalf("setup offered a sign-in:\n%s", stderr)
		}
	})
}

// A7 (row R15): the input closed with the first offer unanswered: the
// session ends within the bound, the warning at most once, no menu.
func TestFirstRunEndOfInputAtOfferEnds(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.noAnswer = true
		h.start(Prompt{})
		waitUntil(t, has(h, firstRunQuestion+" [Y/n]: "))
		if err := endSession(t, h); err != nil {
			t.Fatal(err)
		}
		if n := warningCount(h); n > 1 {
			t.Errorf("the warning was printed %d times", n)
		}
		if strings.Contains(h.stderr.String(), providerMenu) {
			t.Errorf("the provider menu was shown")
		}
	})
}

// A10 (row R61): Enter alone at the first offer, [Y/n], starts a round.
// The input is then closed, which ends the session at every phase.
func TestFirstRunEnterAtFirstOfferSignsIn(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		h.answer = enter
		h.start(Prompt{})
		waitUntil(t, has(h, providerMenu))
		if err := endSession(t, h); err != nil {
			t.Fatal(err)
		}
	})
}

// A11 (row R42): the process's context ends while the key prompt reads:
// the session returns its cause, and the offer was made once.
func TestFirstRunContextEndReturns(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		h, _ := firstRunHarness(t, enter)
		ctx, cancel := context.WithCancelCause(t.Context())
		h.e.ctx = ctx
		cause := errors.New("the run ended")
		h.start(Prompt{})
		toKeyPrompt(t, h)
		cancel(cause)
		if err := h.wait(); !errors.Is(err, cause) {
			t.Fatalf("the session returned %v; want the context's cause", err)
		}
		if n := offerCount(h); n != 1 {
			t.Errorf("the offer was made %d times; want once", n)
		}
	})
}
