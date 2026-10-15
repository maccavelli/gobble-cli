package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/maccavelli/gobble-cli/internal/filelock"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// refreshPolicy is how long a refresh waits for its provider's refresh
// lock: the SDK's own store's 40 s, backing off from 20 ms to 2 s.
var refreshPolicy = filelock.Policy{First: 20 * time.Millisecond, Max: 2 * time.Second, Deadline: 40 * time.Second}

// Logins are auth.json's oauth entries as provider.LoginStore: the file at
// Path, each change under auth.json's lock, and each provider's refreshes
// under its own lock file, never auth.json's, which a save inside a refresh
// takes (0005-PLAN F4-3b decision 4).
type Logins struct {
	Path string
}

var _ provider.LoginStore = Logins{}

// Load is id's oauth entry as a login; nil when id has none, or an entry of
// another type.
func (l Logins) Load(_ context.Context, id string) (*provider.Login, error) {
	f, err := Load(l.Path)
	if err != nil {
		return nil, err
	}
	e, ok := f.Entry(id)
	if !ok || e.Type != TypeOAuth {
		return nil, nil
	}
	login := &provider.Login{
		Access: e.Access, Refresh: e.Refresh,
		Issuer: e.Issuer, ClientID: e.ClientID, AccountID: e.AccountID, TokenURL: e.TokenURL, FedRAMP: e.FedRAMP,
	}
	if e.Expires > 0 {
		login.Expires = time.UnixMilli(e.Expires)
	}
	return login, nil
}

// Save stores login as id's oauth entry, keeping the entry's other fields.
func (l Logins) Save(ctx context.Context, id string, login *provider.Login) error {
	e := Entry{
		Access: login.Access, Refresh: login.Refresh,
		Issuer: login.Issuer, ClientID: login.ClientID, AccountID: login.AccountID, TokenURL: login.TokenURL,
		FedRAMP: login.FedRAMP,
	}
	if !login.Expires.IsZero() {
		e.Expires = login.Expires.UnixMilli()
	}
	return SetOAuth(ctx, l.Path, id, e)
}

// Delete removes id's entry.
func (l Logins) Delete(ctx context.Context, id string) error {
	_, err := Delete(ctx, l.Path, id)
	return err
}

// LockRefresh takes id's refresh lock, auth.json.<id>.refresh.lock beside
// the file, waiting up to 40 s.
func (l Logins) LockRefresh(ctx context.Context, id string) (func(), error) {
	release, err := filelock.Lock(ctx, l.Path+"."+id+".refresh.lock", refreshPolicy)
	if err != nil {
		return nil, fmt.Errorf("auth.json: %s refresh lock: %w", id, err)
	}
	return func() { _ = release() }, nil //nolint:errcheck // the SDK's unlock reports nothing; the OS frees the lock with the file
}
