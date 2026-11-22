package auth

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/maccavelli/gobble-cli/internal/filelock"
	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/jsonx"
)

// planProvider is the provider whose plan entry the migration's steps 4
// and 5 read, and clientsFile the state file of issued clients.
const (
	planProvider = "openai"
	clientsFile  = "clients.json"
)

// dormantFields are the members an oauth entry never holds: one left from an
// earlier api_key entry is a dormant secret beside the tokens (0052-MADR F25).
var dormantFields = []string{"key", "workspace"}

// MigrateOptions are what Migrate reads and the hooks it calls. internal/auth
// may not import the SDK, so the plan session, its answer and the consent's
// write reach it only through PlanSession and CarryConsent, which
// internal/cli builds from llm/provider (0052-MADR D9 step 5, Q2).
type MigrateOptions struct {
	// AuthPath is auth.json's path. A symbolic link there is resolved and its
	// target migrated; the lock stays AuthPath+".lock".
	AuthPath string
	// StateDir is the state directory, which holds host-id and clients.json.
	StateDir string
	// Settings reports whether the global settings.yaml holds
	// chatgptPlanBackground: true, and whether it could be read: (false,
	// false) is a file that cannot be read or parsed. It is asked only when
	// the plan session is unanswered and its entry has no kind member. Nil
	// is (false, true).
	Settings func() (bool, bool)
	// PlanSession reports whether openai's stored session is a ChatGPT plan
	// session, and whether its consent is answered. It is asked only when
	// auth.json has an openai oauth entry. Nil is no plan session.
	PlanSession func(ctx context.Context) (plan, answered bool, err error)
	// CarryConsent records the plan consent granted. It is called after
	// auth.json.lock is released. Nil carries nothing.
	CarryConsent func(ctx context.Context) error
	// Log gives the logger for the migration's records, called only when
	// there is one to write; nil writes none. internal/cli gives a view that
	// writes to the log file alone.
	Log func() *slog.Logger
	// Now gives the backup's time; nil is time.Now.
	Now func() time.Time
}

// Report is what one Migrate did: the providers whose dormant key or
// workspace it dropped, in the file's order; the backup's file name, "" for
// none; and whether it wrote host-id, openai's client record and the consent.
type Report struct {
	Dropped                       []string
	Backup                        string
	HostID, ClientRecord, Consent bool
}

// migrateHooks are test seams, nil in production, set only by internal/auth's
// own tests: afterFirstRead runs between the read without the lock and the
// lock; beforeHostIDCreate between host-id's check and its create;
// beforeClientSave between step 4's check of clients.json and its save;
// failBackup after the backup's bytes are written to its temporary file;
// failAuthWrite before auth.json's write and afterAuthWrite after it. A
// non-nil error from a hook stops the migration with it.
var migrateHooks struct {
	afterFirstRead, beforeHostIDCreate func()
	failBackup, failAuthWrite          func() error
	afterAuthWrite                     func() error
	beforeClientSave                   func()
}

// ErrUnreadable marks Migrate's error for an auth.json that cannot be read or
// parsed: the command that reads the file reports that itself, so a caller
// keeps this error out of the command's output (0052-MADR D9, row M66).
var ErrUnreadable = errors.New("auth.json cannot be read")

// migrateWork is what one read of auth.json leaves to do: the members whose
// dormant key or workspace step 3 drops; whether step 4 has ids to adopt from
// the plan entry, and its ids; whether step 5 has a consent to carry.
type migrateWork struct {
	data             []byte
	file             *File
	drop             []int
	adopt            bool
	hostID, clientID string
	carry            bool
}

func (w *migrateWork) none() bool { return len(w.drop) == 0 && !w.adopt && !w.carry }

// Migrate is auth.json's one-time migration (0052-MADR D9, as the owner's
// answers of 2026-11-21 amend it). It reads the file without the lock and,
// only when that read finds work, takes auth.json.lock, reads again and,
// still finding work: backs the file up when step 3 will change it, through
// an owner-only temporary file renamed to auth.json.bak-<UTC> beside it
// after a check that the name is free; drops each oauth entry's dormant key
// and workspace, keeping every other member's bytes and writing the top
// level as gobble does; adopts the plan entry's host id and client into the
// state directory; then, the lock released, carries the plan consent through
// CarryConsent. The context is checked before each step; a write already
// begun completes. A file that cannot be read is an error, with nothing
// written. Each later run finds nothing to do and writes nothing.
func Migrate(ctx context.Context, o MigrateOptions) (Report, error) {
	var r Report
	if err := ctx.Err(); err != nil {
		return r, err
	}
	target, err := migrateTarget(o.AuthPath)
	if err != nil {
		return r, err
	}
	w, err := o.work(ctx, target)
	if err != nil || w.none() {
		return r, err
	}
	if h := migrateHooks.afterFirstRead; h != nil {
		h()
	}
	w, err = o.locked(ctx, target, &r)
	if err != nil || !w.carry {
		return r, err
	}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if err := o.CarryConsent(ctx); err != nil {
		return r, fmt.Errorf("auth.json: carry the plan consent: %w", err)
	}
	r.Consent = true
	return r, nil
}

// migrateTarget is the file a symbolic link at path names, or path itself
// when nothing is there (0052-MADR D9, Q7).
func migrateTarget(path string) (string, error) {
	target, err := filepath.EvalSymlinks(path)
	if errors.Is(err, fs.ErrNotExist) {
		return path, nil
	}
	if err != nil {
		return "", fmt.Errorf("auth.json: %w", err)
	}
	return target, nil
}

// locked is steps 2 to 4 under auth.json.lock: the work computed again from a
// read under the lock, then the backup, step 3's write and step 4. The lock
// is released before it returns, so before any carry.
func (o MigrateOptions) locked(ctx context.Context, target string, r *Report) (w *migrateWork, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	release, err := filelock.Lock(ctx, o.AuthPath+".lock", authPolicy)
	if err != nil {
		return nil, fmt.Errorf("auth.json: %w", err)
	}
	defer func() { err = errors.Join(err, release()) }()
	if w, err = o.work(ctx, target); err != nil || w.none() {
		return w, err
	}
	if len(w.drop) > 0 {
		if r.Backup, err = o.backup(target, w.data); err != nil {
			return w, err
		}
		if err := ctx.Err(); err != nil {
			return w, err
		}
		if err := o.dropDormant(ctx, target, w, r); err != nil {
			return w, err
		}
	}
	if w.adopt {
		if err := ctx.Err(); err != nil {
			return w, err
		}
		if err := o.adopt(ctx, w, r); err != nil {
			return w, err
		}
	}
	return w, nil
}

// work reads target and computes what the migration has to do.
func (o MigrateOptions) work(ctx context.Context, target string) (*migrateWork, error) {
	data, err := fsx.ReadRegularFile(target, fsx.ConfigLimit)
	if errors.Is(err, fs.ErrNotExist) {
		return &migrateWork{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	f, err := parse(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnreadable, target, err)
	}
	w := &migrateWork{data: data, file: f}
	var planHasKind bool
	for i, m := range f.members {
		if m.entry.Type != TypeOAuth {
			continue
		}
		names, err := fieldNames(m.value)
		if err != nil {
			return nil, fmt.Errorf("auth.json: provider %s: %w", m.provider, err)
		}
		if names["key"] || names["workspace"] {
			w.drop = append(w.drop, i)
		}
		if m.provider == planProvider {
			planHasKind = names["kind"]
		}
	}
	e, ok := f.Entry(planProvider)
	if !ok || e.Type != TypeOAuth || o.PlanSession == nil {
		return w, nil
	}
	plan, answered, err := o.PlanSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth.json: %s's session: %w", planProvider, err)
	}
	if !plan {
		return w, nil
	}
	w.hostID, w.clientID = e.HostID, e.ClientID
	w.adopt = o.adoptWork(ctx, w)
	if answered || planHasKind || o.Settings == nil {
		return w, nil
	}
	switch carry, ok := o.Settings(); {
	case !ok:
		return nil, errors.New("settings.yaml cannot be read, so whether to carry the ChatGPT plan consent is not known; auth.json is not migrated")
	case carry:
		w.carry = true
	}
	return w, nil
}

// adoptWork reports whether step 4 has work: the plan entry holds a hostId,
// and host-id lacks it, or the client record is missing while host-id holds
// it. A state file that cannot be read counts as work, so that step 4 meets
// it and reports it.
func (o MigrateOptions) adoptWork(ctx context.Context, w *migrateWork) bool {
	if w.hostID == "" {
		return false
	}
	have, err := readHostID(filepath.Join(o.StateDir, hostIDFile))
	if err != nil || have == "" {
		return true
	}
	if have != w.hostID || w.clientID == "" {
		return false
	}
	rec, err := Clients{Path: filepath.Join(o.StateDir, clientsFile)}.Load(ctx, planProvider)
	return err != nil || rec == (ClientRecord{})
}

// fieldNames are the names of the members of the object v.
func fieldNames(v jsontext.Value) (map[string]bool, error) {
	d := jsontext.NewDecoder(bytes.NewReader(v), jsonx.StrictOptions())
	if _, err := d.ReadToken(); err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for d.PeekKind() != '}' {
		tok, err := d.ReadToken()
		if err != nil {
			return nil, err
		}
		names[tok.String()] = true
		if err := d.SkipValue(); err != nil {
			return nil, err
		}
	}
	return names, nil
}

// backup writes data, auth.json's bytes as read under the lock, to a
// temporary file beside target, owner-only before its first byte and synced,
// and renames it to auth.json.bak-<UTC> after a check that no file has that
// name: a taken name is an error. The temporary file is removed on any
// failure (0052-MADR D9 step 2, Q4).
func (o MigrateOptions) backup(target string, data []byte) (name string, err error) {
	now := time.Now
	if o.Now != nil {
		now = o.Now
	}
	dir := filepath.Dir(target)
	name = "auth.json.bak-" + now().UTC().Format("20060102T150405Z")
	tmp, err := os.CreateTemp(dir, ".auth.json.bak-*.tmp")
	if err != nil {
		return "", fmt.Errorf("auth.json: backup: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			if rerr := os.Remove(tmpName); rerr != nil && !errors.Is(rerr, fs.ErrNotExist) {
				err = errors.Join(err, rerr)
			}
		}
	}()
	if err := writeTemp(tmp, data); err != nil {
		return "", fmt.Errorf("auth.json: backup: %w", err)
	}
	if h := migrateHooks.failBackup; h != nil {
		if err := h(); err != nil {
			return "", fmt.Errorf("auth.json: backup: %w", err)
		}
	}
	dst := filepath.Join(dir, name)
	if _, err := os.Lstat(dst); err == nil {
		return "", fmt.Errorf("auth.json: backup: %s already exists; auth.json is not migrated", name)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("auth.json: backup: %w", err)
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return "", fmt.Errorf("auth.json: backup: %w", err)
	}
	return name, nil
}

// dropDormant is step 3: each member in w.drop without its key and workspace,
// laid out as gobble's writes lay out an edited entry, written with every
// other member's bytes as they stand; then one Info record per provider,
// naming it and the backup, never a value (0052-MADR D9 step 3, Q5).
func (o MigrateOptions) dropDormant(ctx context.Context, target string, w *migrateWork, r *Report) error {
	var dropped []string
	for _, i := range w.drop {
		m := &w.file.members[i]
		v, err := merge(m.value, nil, dormantFields)
		if err != nil {
			return fmt.Errorf("auth.json: provider %s: %w", m.provider, err)
		}
		m.value = indented(v)
		dropped = append(dropped, m.provider)
	}
	if h := migrateHooks.failAuthWrite; h != nil {
		if err := h(); err != nil {
			return fmt.Errorf("auth.json: %w", err)
		}
	}
	if err := write(target, w.file.encodeKept()); err != nil {
		return err
	}
	r.Dropped = dropped
	if o.Log != nil {
		if log := o.Log(); log != nil {
			for _, p := range dropped {
				log.InfoContext(ctx, "auth.json: dropped a dormant key or workspace",
					slog.String("provider", p), slog.String("backup", r.Backup))
			}
		}
	}
	if h := migrateHooks.afterAuthWrite; h != nil {
		return h()
	}
	return nil
}

// encodeKept is the file in gobble's layout, LF and two spaces deep with a
// final newline, as encode writes it, but with every member's bytes as they
// stand, a one-line member included, where encode re-indents one.
func (f *File) encodeKept() []byte {
	var b bytes.Buffer
	b.WriteString("{")
	for i, m := range f.members {
		if i > 0 {
			b.WriteString(",")
		}
		name, _ := jsontext.AppendQuote(nil, m.provider) //nolint:errcheck // a name read from JSON quotes
		fmt.Fprintf(&b, "\n  %s: %s", name, m.value)
	}
	if len(f.members) > 0 {
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return b.Bytes()
}

// adopt is step 4: the plan entry's hostId placed in host-id when that file
// is absent or empty, and its client in clients.json when the entry holds
// both ids, host-id then holds its hostId, and clients.json has no openai
// record. A hostId that is not a urn:uuid, or a host-id that holds anything
// but one, is an error, and nothing is adopted (0052-MADR D9 step 4, Q3).
func (o MigrateOptions) adopt(ctx context.Context, w *migrateWork, r *Report) error {
	if !validHostID(w.hostID) {
		return fmt.Errorf("auth.json: %s's hostId is not a urn:uuid; its ids are not adopted", planProvider)
	}
	got, wrote, err := placeHostIDAfter(ctx, o.StateDir, w.hostID, migrateHooks.beforeHostIDCreate)
	if err != nil {
		return err
	}
	r.HostID = wrote
	if w.clientID == "" || got != w.hostID {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	wrote, err = adoptClient(ctx, Clients{Path: filepath.Join(o.StateDir, clientsFile)}, ClientRecord{ClientID: w.clientID, HostID: w.hostID})
	if err != nil {
		return err
	}
	r.ClientRecord = wrote
	return nil
}

// adoptClient saves rec as openai's record in c when, under
// clients.json.lock, c has none, reporting whether it did; a record another
// process saved after step 4's check is kept (row M67). A clients.json that
// cannot be read is an error and is not replaced.
func adoptClient(ctx context.Context, c Clients, rec ClientRecord) (bool, error) {
	if h := migrateHooks.beforeClientSave; h != nil {
		h()
	}
	return c.saveIfAbsent(ctx, planProvider, rec)
}
