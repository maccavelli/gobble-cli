package provider_test

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"

	gauth "github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// keyOf is a JSON member name as a field name compares: lower case, no
// underscores ("client_id" is ClientID's, "fedramp" FedRAMP's).
func keyOf(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", "")) }

// Every field of the SDK's session that the SDK stores survives gobble's
// chain: loginOf, Logins.Save, auth.json, Logins.Load and session. The
// fields are read by reflection, each set to a value of its own, so a field
// the SDK adds and stores fails here until gobble carries it (0052-PLAN P2-2).
//
// Which fields are stored is the SDK's to say, not a list kept here: the
// session is saved by the SDK's own FileTokenStore, and a field is stored
// when its member is in the file. The collaborators a session holds (its
// Store, HTTPClient and Logger, which are interfaces and pointers) hold no
// data and are not filled. Any other exported field of a kind this test
// cannot fill fails it.
func TestLoginCarriesEverySessionField(t *testing.T) {
	const id = "openai" // the key the session is stored under: its Provider
	var in auth.OAuthSession
	v := reflect.ValueOf(&in).Elem()
	var filled []string
	for i := range v.NumField() {
		f, sf := v.Field(i), v.Type().Field(i)
		if !sf.IsExported() {
			continue
		}
		switch {
		case sf.Name == "Provider":
			f.SetString(id)
		case f.Kind() == reflect.Interface || f.Kind() == reflect.Pointer:
			continue
		case f.Kind() == reflect.String:
			f.SetString("v" + sf.Name)
		case f.Kind() == reflect.Bool:
			f.SetBool(true)
		case f.Type() == reflect.TypeFor[time.Time]():
			f.Set(reflect.ValueOf(time.UnixMilli(1781000000000 + int64(i)*1000)))
		default:
			t.Fatalf("OAuthSession.%s is a %s, which this test cannot fill: teach it, or the SDK stores what gobble cannot carry", sf.Name, f.Type())
		}
		filled = append(filled, sf.Name)
	}

	// The SDK's own store says which of them it stores.
	sdkDir := t.TempDir()
	fs, err := auth.NewFileTokenStore(sdkDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.Save(t.Context(), in.Provider, &in); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(sdkDir, "*.json"))
	if err != nil || len(files) != 1 {
		t.Fatalf("the SDK's store wrote %v, %v; want one file", files, err)
	}
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var members map[string]any
	if err := jsonx.Unmarshal(raw, &members); err != nil {
		t.Fatal(err)
	}
	stored := map[string]bool{}
	for k := range members {
		stored[keyOf(k)] = true
	}
	var carried []string
	for _, name := range filled {
		if stored[keyOf(name)] {
			carried = append(carried, name)
			delete(stored, keyOf(name))
		}
	}
	for k := range stored {
		t.Errorf("the SDK stores member %q, which matches no OAuthSession field this test filled", k)
	}
	// The reflection must not pass by finding nothing: the two this phase
	// adds are in it.
	for _, must := range []string{"Kind", "Consents", "Access", "HostID"} {
		if !slices.Contains(carried, must) {
			t.Fatalf("OAuthSession.%s is not among the stored fields %v", must, carried)
		}
	}

	// The chain, through the real store and a real auth.json.
	logins := gauth.Logins{Path: filepath.Join(t.TempDir(), "auth.json")}
	if err := logins.Save(t.Context(), id, provider.LoginOf(&in)); err != nil {
		t.Fatal(err)
	}
	got, err := logins.Load(t.Context(), id)
	if err != nil || got == nil {
		t.Fatalf("Load: %v, %v", got, err)
	}
	out := reflect.ValueOf(provider.SessionOf(got, id)).Elem()
	for _, name := range carried {
		want, have := v.FieldByName(name).Interface(), out.FieldByName(name).Interface()
		if wt, ok := want.(time.Time); ok {
			if !wt.Equal(have.(time.Time)) {
				t.Errorf("OAuthSession.%s: set %v, came back %v", name, want, have)
			}
			continue
		}
		if want != have {
			t.Errorf("OAuthSession.%s: set %v, came back %v", name, want, have)
		}
	}
}
