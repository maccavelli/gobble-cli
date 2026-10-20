package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/appdirs"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// recentKept is how many recent models models.json keeps (0025-MADR D3).
const recentKept = 10

// lastGoodLimit is how old a kept listing may be to be shown, the SDK's own
// limit for WithLastGood (0041-MADR D6 of go-llmprovider-sdk).
const lastGoodLimit = 24 * time.Hour

// providerChoices are every key provider, in registry order, with its state
// and models (0025-MADR D4). run is the run's provider, whose models are
// runModels and whose listing a session waits for; the others are read as
// they stand, a listing still running shown by its kept listing, stale, or
// its static models.
func (e *runEnv) providerChoices(run string, runModels []string) []acpserver.ProviderChoice {
	var out []acpserver.ProviderChoice
	for _, id := range provider.KeyProviders() {
		p := acpserver.ProviderChoice{ID: id, Label: provider.Label(id)}
		if !e.hasCredential(id) {
			p.State = acpserver.StateNoCredential
			p.Reason = fmt.Sprintf("— set %s, or gobble auth login %s", provider.CredentialVar(id), id)
			out = append(out, p)
			continue
		}
		l := e.modelListing(provider.Selection{ID: id})
		if id == run {
			p.State, p.Reason, p.Age = listingState(id, l.get(), time.Now())
			p.Models = runModels
			out = append(out, p)
			continue
		}
		r, ok := l.peek()
		switch {
		case ok:
			p.State, p.Reason, p.Age = listingState(id, r, time.Now())
			if p.State.Usable() {
				p.Models = r.Models
			}
		case len(l.kept.models) > 0:
			p.State, p.Models, p.Age = acpserver.StateStale, l.kept.models, time.Since(l.kept.at)
		default:
			p.State, p.Models = acpserver.StateListing, provider.Models(id)
		}
		out = append(out, p)
	}
	return out
}

// listingState is a listing's state, reason and age: a live listing's, a
// stale one's with its age, a public one's not checked; a failed one's by
// its error (0025-MADR D4, deviation 1).
func listingState(id string, r provider.Listing, now time.Time) (acpserver.ProviderState, string, time.Duration) {
	switch {
	case r.Live && r.Source == "stale":
		return acpserver.StateStale, "", now.Sub(r.FetchedAt)
	case r.Live && provider.PublicListing(id):
		return acpserver.StateNotChecked, "", 0
	case r.Live:
		return acpserver.StateLive, "", 0
	}
	state, reason := acpserver.StateOf(id, r.Err)
	return state, reason, 0
}

// recentFile is models.json's shape.
type recentFile struct {
	Recent []string `json:"recent"`
}

// recentMu guards the recent models, read once from models.json, and its
// writes (0025-MADR D3).
var recentMu sync.Mutex

// recentModels are the recent models, newest first: models.json's, read
// once a process; an unreadable file is none, with a warning.
func (e *runEnv) recentModels() []string {
	recentMu.Lock()
	defer recentMu.Unlock()
	return slices.Clone(e.loadRecent())
}

// loadRecent reads models.json once; recentMu is held.
func (e *runEnv) loadRecent() []string {
	if e.recentLoaded {
		return e.recent
	}
	e.recentLoaded = true
	path, err := stateFile("models.json")
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(path) //nolint:gosec // G304: the state directory's own models.json
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	var f recentFile
	if err == nil {
		err = json.Unmarshal(b, &f)
	}
	if err != nil {
		e.out.Warnf("models.json: %v; the recent models are not read", err)
		return nil
	}
	e.recent = f.Recent
	return e.recent
}

// rememberModel puts ref first among the recent models, ten kept, each
// once, and writes models.json owner-only; a failed write is a warning.
func (e *runEnv) rememberModel(ref string) {
	recentMu.Lock()
	defer recentMu.Unlock()
	recent := slices.DeleteFunc(slices.Clone(e.loadRecent()), func(r string) bool { return r == ref })
	recent = append([]string{ref}, recent...)
	if len(recent) > recentKept {
		recent = recent[:recentKept]
	}
	e.recent = recent
	path, err := stateFile("models.json")
	if err == nil {
		var b []byte
		if b, err = json.MarshalIndent(recentFile{Recent: recent}, "", "  "); err == nil {
			err = writePrivate(path, append(b, '\n'))
		}
	}
	if err != nil {
		e.out.Warnf("models.json: %v", err)
	}
}

// stateFile is name in the state directory, which it creates owner-only.
func stateFile(name string) (string, error) {
	dirs, _, err := systemDirs()
	if err != nil {
		return "", err
	}
	if err := appdirs.EnsurePrivateDir(dirs.State); err != nil {
		return "", err
	}
	return filepath.Join(dirs.State, name), nil
}

// keptListing is a provider's listing kept on disk: its models, its usable
// ids for WithLastGood, and when it was read (0025-MADR D4).
type keptListing struct {
	Models    []string  `json:"models"`
	Usable    []string  `json:"usable"`
	FetchedAt time.Time `json:"fetchedAt"`
}

// kept is a listing read from disk, for display and WithLastGood.
type kept struct {
	models, usable []string
	at             time.Time
}

// catalogFile is <cache>/catalog/<id>.json, its directory created.
func catalogFile(id string) (string, error) {
	dirs, _, err := systemDirs()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dirs.Cache, "catalog")
	if err := appdirs.EnsurePrivateDir(dir); err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

// readKept is provider id's kept listing, when one exists within a day.
func readKept(id string, now time.Time) kept {
	path, err := catalogFile(id)
	if err != nil {
		return kept{}
	}
	b, err := os.ReadFile(path) //nolint:gosec // G304: the cache directory's own catalog/<id>.json
	if err != nil {
		return kept{}
	}
	var k keptListing
	if json.Unmarshal(b, &k) != nil || k.FetchedAt.IsZero() || now.Sub(k.FetchedAt) > lastGoodLimit {
		return kept{}
	}
	return kept{models: k.Models, usable: k.Usable, at: k.FetchedAt}
}

// writeKept keeps a live listing on disk for the next run; a failure is
// not reported, since the listing itself succeeded.
func writeKept(id string, r provider.Listing) {
	path, err := catalogFile(id)
	if err != nil {
		return
	}
	b, err := json.Marshal(keptListing{Models: r.Models, Usable: r.Usable, FetchedAt: r.FetchedAt})
	if err != nil {
		return
	}
	_ = writePrivate(path, b) //nolint:errcheck // a kept listing is a convenience
}

// writePrivate replaces path with data through a temporary file in its
// directory, which os.CreateTemp opens owner-only.
func writePrivate(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		return errors.Join(err, tmp.Close(), os.Remove(name))
	}
	if err := tmp.Close(); err != nil {
		return errors.Join(err, os.Remove(name))
	}
	if err := os.Rename(name, path); err != nil {
		return errors.Join(err, os.Remove(name))
	}
	return nil
}
