package cli

import (
	"encoding/json/jsontext"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
)

// observationFileName is the output caps services stated, kept for a day, in
// the state directory beside reasoning-refusals.json (0066-MADR D7).
const observationFileName = "limit-observations.json"

// observationsKept is how many caps one provider, base URL and scope keep: the
// bound reasoning-refusals.json keeps (refusalsKept; 0066-MADR D7).
const observationsKept = refusalsKept

// observationAge is how long a saved cap is handed back: a day, as the SDK
// reads a remembered cap (0066-MADR D7, S14).
const observationAge = 24 * time.Hour

// observationSkew is how far after now a saved cap's time may be and still be
// read: a clock step between its write and this read. One seen later is
// dropped, as an expired one is (0071-PLAN's review deviation of 2026-11-25,
// R2).
const observationSkew = 5 * time.Minute

// savedObservation is one saved cap: the provider it was stated by, then
// what the SDK's observation holds (0071-MADR P15, P26).
type savedObservation struct {
	Provider string    `json:"provider"`
	Model    string    `json:"model"`
	BaseURL  string    `json:"baseURL,omitzero"`
	Scope    string    `json:"scope,omitzero"`
	Tokens   int       `json:"tokens"`
	Seen     time.Time `json:"seen"`
}

// observationFile is limit-observations.json's shape: the caps, newest first.
type observationFile struct {
	Observations []savedObservation `json:"observations"`
}

// observationMu guards the saved caps, read once, and their writes.
var observationMu sync.Mutex

// providerObservations are provider id's saved caps, newest first, for its
// provider, lookups and discovery to hand the SDK as a layer; another
// provider's are never handed (0071-MADR P24, P26; 0066-MADR S14).
func (e *runEnv) providerObservations(id string) []llm.LimitObservation {
	observationMu.Lock()
	defer observationMu.Unlock()
	var out []llm.LimitObservation
	for _, o := range e.loadObservations() {
		if o.Provider == id {
			out = append(out, llm.LimitObservation{Model: o.Model, BaseURL: o.BaseURL, Scope: o.Scope, Tokens: o.Tokens, Seen: o.Seen})
		}
	}
	return out
}

// loadObservations reads limit-observations.json once; observationMu is
// held. An unreadable file is none, with one warning (0066-MADR S13); an
// entry with no provider, no model, a cap below 1 or no time is dropped, the
// good ones kept, with one warning for them all (0071-MADR P27).
func (e *runEnv) loadObservations() []savedObservation {
	if e.observationsLoaded {
		return e.observations
	}
	e.observationsLoaded = true
	e.observations = nil
	path, err := stateFile(observationFileName)
	if err != nil {
		return nil
	}
	list, bad, err := readObservations(path, time.Now())
	if err != nil {
		e.out.Warnf("%s: %v; the saved output caps are not read", observationFileName, err)
	}
	if bad > 0 {
		e.out.Warnf("%s: %d saved output caps lack a provider, a model, a cap or a time; they are not read", observationFileName, bad)
	}
	e.observations = list
	return e.observations
}

// readObservations is the fresh caps the limit-observations.json at path
// holds now: seen less than observationAge before now and not more than
// observationSkew after it; none when there is no file. bad counts the
// entries with no provider, no model, a cap below 1 or no time, which are not
// read; err is why the file could not be read (0066-MADR S13, S14; 0071-MADR
// P27; 0071-PLAN's review deviation of 2026-11-25, R2).
func readObservations(path string, now time.Time) (list []savedObservation, bad int, err error) {
	b, err := os.ReadFile(path) //nolint:gosec // G304: the state directory's own limit-observations.json
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil
	}
	var f observationFile
	if err == nil {
		err = jsonx.UnmarshalStrict(b, &f)
	}
	if err != nil {
		return nil, 0, err
	}
	for _, o := range f.Observations {
		switch {
		case o.Provider == "" || strings.TrimSpace(o.Model) == "" || o.Tokens < 1 || o.Seen.IsZero():
			bad++
		case now.Sub(o.Seen) < observationAge && o.Seen.Sub(now) <= observationSkew:
			list = append(list, o)
		}
	}
	return list, bad, nil
}

// rememberObservation puts the cap provider id's service stated first, in
// place of any earlier one for the same model, base URL and scope unless that
// one is newer, ten kept per provider, base URL and scope, and writes the
// file owner-only. A failed write is a warning (0066-MADR D7, S12; 0071-MADR
// P19).
func (e *runEnv) rememberObservation(id string, o llm.LimitObservation) {
	if id == "" || strings.TrimSpace(o.Model) == "" || o.Tokens < 1 || o.Seen.IsZero() {
		return
	}
	saved := savedObservation{Provider: id, Model: o.Model, BaseURL: o.BaseURL, Scope: o.Scope, Tokens: o.Tokens, Seen: o.Seen}
	observationMu.Lock()
	defer observationMu.Unlock()
	e.loadObservations()
	e.updateObservations(func(list []savedObservation) []savedObservation {
		if slices.ContainsFunc(list, func(old savedObservation) bool { return sameKey(old, saved) && old.Seen.After(saved.Seen) }) {
			return list // the newest Seen wins
		}
		list = slices.DeleteFunc(list, func(old savedObservation) bool { return sameKey(old, saved) })
		list = append([]savedObservation{saved}, list...)
		kept := 0
		return slices.DeleteFunc(list, func(old savedObservation) bool {
			if !sameScope(old, saved) {
				return false
			}
			kept++
			return kept > observationsKept
		})
	})
}

// dropExceeded drops provider id's saved caps of model at its base URL and
// scope that a completed call's output tokens exceed: the service answered
// past them, so they are wrong, as the SDK drops its memo's (0071-MADR P25;
// go-llmprovider-sdk 0089-MADR DL39). The file is re-read and written only
// when the run's list holds such a cap.
func (e *runEnv) dropExceeded(id, model string, output int) {
	if output < 1 {
		return
	}
	base, scope := e.observationScope(id)
	exceeded := func(o savedObservation) bool {
		return o.Provider == id && strings.EqualFold(o.Model, model) && trimBase(o.BaseURL) == base &&
			strings.TrimSpace(o.Scope) == scope && o.Tokens < output
	}
	observationMu.Lock()
	defer observationMu.Unlock()
	if !slices.ContainsFunc(e.loadObservations(), exceeded) {
		return
	}
	e.updateObservations(func(list []savedObservation) []savedObservation { return slices.DeleteFunc(list, exceeded) })
}

// updateObservations applies change to the caps limit-observations.json holds
// now, read afresh, and writes the result, which becomes the run's list;
// observationMu is held. Another process's writes since this run read the
// file are kept, and an entry it dropped stays dropped (0066-MADR D7;
// 0071-PLAN's review deviation of 2026-11-25, R5). An unreadable file is
// none: the load warned of it once (S13), and the write replaces it.
func (e *runEnv) updateObservations(change func([]savedObservation) []savedObservation) {
	var list []savedObservation
	if path, err := stateFile(observationFileName); err == nil {
		if fresh, _, err := readObservations(path, time.Now()); err == nil {
			list = fresh
		}
	}
	e.writeObservations(change(list))
}

// observationScope is the base URL and scope provider id's calls run in, as
// an observation names them: models.json's base URL, else the default's
// empty one; the stored key's workspace, which only a Claude key carries,
// else none: gobble passes no Kilo organization (0071-MADR F25).
func (e *runEnv) observationScope(id string) (base, scope string) {
	return trimBase(e.modelsJSON().BaseURL(id)), strings.TrimSpace(e.knownWorkspace(id))
}

// writeObservations keeps list and writes it; observationMu is held.
func (e *runEnv) writeObservations(list []savedObservation) {
	e.observations = list
	path, err := stateFile(observationFileName)
	if err == nil {
		var b []byte
		if b, err = jsonx.MarshalStrict(observationFile{Observations: list}, jsontext.WithIndent("  ")); err == nil {
			err = writePrivate(path, append(b, '\n'))
		}
	}
	if err != nil {
		e.out.Warnf("%s: %v", observationFileName, err)
	}
}

// sameScope reports that a and b were stated by one provider at one base URL
// in one scope.
func sameScope(a, b savedObservation) bool {
	return a.Provider == b.Provider && trimBase(a.BaseURL) == trimBase(b.BaseURL) && strings.TrimSpace(a.Scope) == strings.TrimSpace(b.Scope)
}

// sameKey reports that a and b are one cap's entries: one provider, base URL,
// scope and model.
func sameKey(a, b savedObservation) bool {
	return sameScope(a, b) && strings.EqualFold(a.Model, b.Model)
}

// trimBase is a base URL as an observation names it: without spaces or
// trailing slashes; empty is the provider's default.
func trimBase(base string) string { return strings.TrimRight(strings.TrimSpace(base), "/") }
