package cli

import (
	"encoding/json/jsontext"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
)

// refusalFileName is the learned reasoning refusals' file in the state
// directory, beside recent-models.json (0036-MADR D4).
const refusalFileName = "reasoning-refusals.json"

// refusalsKept is how many refusals one provider's list keeps. It is the
// same bound recent-models.json keeps (recentKept).
const refusalsKept = 10

// refusalLists are the saved refusals, one list per provider.
type refusalLists map[string][]llm.ReasoningRefusal

// refusalFile is reasoning-refusals.json's shape.
type refusalFile struct {
	Providers refusalLists `json:"providers"`
}

// refusalMu guards the learned refusals, read once, and their writes.
var refusalMu sync.Mutex

// providerRefusals are provider id's saved refusals. An unreadable file is
// none, with one warning: the provider is still built, and the SDK relearns
// (0036-MADR D4).
func (e *runEnv) providerRefusals(id string) []llm.ReasoningRefusal {
	refusalMu.Lock()
	defer refusalMu.Unlock()
	return slices.Clone(e.loadRefusals()[id])
}

// loadRefusals reads reasoning-refusals.json once; refusalMu is held.
func (e *runEnv) loadRefusals() refusalLists {
	if e.refusalsLoaded {
		return e.refusals
	}
	e.refusalsLoaded = true
	e.refusals = refusalLists{}
	path, err := stateFile(refusalFileName)
	if err != nil {
		return e.refusals
	}
	b, err := os.ReadFile(path) //nolint:gosec // G304: the state directory's own reasoning-refusals.json
	if errors.Is(err, os.ErrNotExist) {
		return e.refusals
	}
	var f refusalFile
	if err == nil {
		err = jsonx.UnmarshalStrict(b, &f)
	}
	if err != nil {
		e.out.Warnf("%s: %v; the reasoning refusals are not read", refusalFileName, err)
		return e.refusals
	}
	if f.Providers != nil {
		e.refusals = f.Providers
	}
	return e.refusals
}

// rememberRefusal puts r first in provider id's list, once, ten kept, and
// writes the file owner-only. A failed write is a warning.
func (e *runEnv) rememberRefusal(id string, r llm.ReasoningRefusal) {
	if id == "" || r.Model == "" || (r.Effort == "") == (r.Shape == "") {
		return
	}
	refusalMu.Lock()
	defer refusalMu.Unlock()
	lists := e.loadRefusals()
	list := slices.DeleteFunc(slices.Clone(lists[id]), sameRefusal(r))
	list = append([]llm.ReasoningRefusal{r}, list...)
	if len(list) > refusalsKept {
		list = list[:refusalsKept]
	}
	lists[id] = list
	e.refusals = lists
	path, err := stateFile(refusalFileName)
	if err == nil {
		var b []byte
		if b, err = jsonx.MarshalStrict(refusalFile{Providers: lists}, jsontext.WithIndent("  ")); err == nil {
			err = writePrivate(path, append(b, '\n'))
		}
	}
	if err != nil {
		e.out.Warnf("%s: %v", refusalFileName, err)
	}
}

// sameRefusal reports that old is r: the same model, effort and shape.
func sameRefusal(r llm.ReasoningRefusal) func(llm.ReasoningRefusal) bool {
	return func(old llm.ReasoningRefusal) bool {
		return strings.EqualFold(old.Model, r.Model) && old.Effort == r.Effort && old.Shape == r.Shape
	}
}
