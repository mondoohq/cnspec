// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package delivery

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
	_ "go.mondoo.com/mql/vault/keyring"
)

// Saving a credential puts it in the operating system's own store -- Keychain
// on macOS, Credential Manager on Windows, Secret Service, KWallet or pass on
// Linux -- and leaves only an id behind in the inventory. The secret then never
// touches a file the launcher writes.
//
// Two limits are worth knowing, both of them in the vault API rather than here:
//
//   - There is no way to list what a vault holds. vault.Vault is Get/Set/Delete
//     and nothing else, so the launcher cannot show you your saved credentials.
//     It keeps its own index of the ids it created, which is why savedIndex
//     exists; anything saved by other means has to be referenced by typing its
//     id.
//   - Delete is not implemented for the keyring backend, so a credential saved
//     here is removed with the OS's own tooling, not from cnspec.

// vaultService is the keychain service name secrets are filed under. It matches
// the name used throughout the cnspec vault documentation, so a credential
// saved here is visible to a hand-written inventory too.
const vaultService = "mondoo-client-vault"

// keyringVault opens the OS keychain.
func keyringVault() (vault.Vault, error) {
	return vault.New(&vault.VaultConfiguration{
		Name: vaultService,
		Type: vault.VaultType_KeyRing,
	})
}

// StoreFunc is the keychain write itself. StoreCredentialWithin takes one
// rather than reaching for StoreCredential directly, because the failure path
// matters more than the success one and cannot be provoked otherwise: the
// launcher passes its own replaceable variable, and a test passes a write that
// never answers.
type StoreFunc = func(id string, cred *vault.Credential) error

// KeychainTimeout is how long the launcher waits for the OS keychain.
//
// A locked keychain does not fail, it asks: macOS raises an authentication
// dialog and Set does not return until it is answered. The launcher is drawing
// a full-screen UI at that moment, so the dialog and the launcher are two
// programs competing for one screen, and a user who does not notice the dialog
// sees a frozen scan. Long enough to type a password, short enough that a
// dialog nobody is looking at does not hold the scan for ever.
const KeychainTimeout = 30 * time.Second

// StoreCredentialWithin is the keychain write with a bound on how long it may
// take, whatever the backend does with the context it is given.
//
// The context passed into vault.Set below is the polite half of this and is not
// enough on its own: the keyring backends reach into the OS through cgo and
// blocking system calls that no context can interrupt, so a timeout that only
// lives in the context would be a timeout that never fires. Waiting on a
// channel instead is a bound that holds regardless.
//
// The write is left running when the wait gives up. It has nowhere else to go,
// and its two outcomes are both harmless: it either lands, leaving a keychain
// entry the inventory did not reference, or it fails. The caller has already
// fallen back to the 0600 inventory by then and has told the user so.
func StoreCredentialWithin(limit time.Duration, write StoreFunc, id string, cred *vault.Credential) error {
	// The write is passed in rather than read from a variable here, and the
	// caller reads its own before calling: the write is left running when the
	// wait gives up, so a goroutine that read a variable could still be reading
	// it after a test had put the real one back.
	done := make(chan error, 1)
	go func() { done <- write(id, cred) }()

	timer := time.NewTimer(limit)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		return errors.New("the OS keychain did not answer within " + limit.String() +
			" — it may be locked and waiting on a dialog behind this window")
	}
}

// StoreCredential writes a credential to the OS keychain under id.
//
// The encoding is not a free choice: the keyring backend rejects anything but
// JSON on write and reports JSON on read regardless of what went in, so a
// secret stored any other way cannot be read back as a credential.
func StoreCredential(id string, cred *vault.Credential) error {
	v, err := keyringVault()
	if err != nil {
		return errors.Wrap(err, "cannot open the OS keychain")
	}
	secret, err := vault.NewSecret(cred, vault.SecretEncoding_encoding_json)
	if err != nil {
		return errors.Wrap(err, "cannot encode the credential")
	}
	secret.Key = id

	// Bounded rather than context.Background(): a backend that does honour a
	// context should give up here rather than hold the launcher open, and one
	// that does not is caught by StoreCredentialWithin instead.
	ctx, cancel := context.WithTimeout(context.Background(), KeychainTimeout)
	defer cancel()
	if _, err := v.Set(ctx, secret); err != nil {
		return errors.Wrap(err, "cannot save the credential to the OS keychain")
	}
	return nil
}

// SavedEntry is one credential the launcher put in the keychain. It records
// where the credential belongs, never what it is.
type SavedEntry struct {
	ID        string `json:"id"`
	Connector string `json:"connector"`
	Label     string `json:"label"`
	SavedAt   string `json:"saved_at"`
}

// savedIndexPath is the launcher's own record of ids it created, which stands
// in for the listing the vault API does not provide.
func savedIndexPath() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, ".config", "mondoo", "cnspec-ui-credentials.json")
}

func loadSavedIndex() []SavedEntry {
	path := savedIndexPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []SavedEntry
	if json.Unmarshal(data, &out) != nil {
		return nil
	}
	return out
}

// RecordSaved adds an entry to the index, replacing any entry with the same id.
func RecordSaved(e SavedEntry) error {
	path := savedIndexPath()
	if path == "" {
		return errors.New("cannot locate the cnspec config directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return errors.Wrap(err, "cannot create the cnspec config directory")
	}

	entries := loadSavedIndex()
	out := entries[:0]
	for _, existing := range entries {
		if existing.ID != e.ID {
			out = append(out, existing)
		}
	}
	out = append(out, e)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return errors.Wrap(err, "cannot encode the credential index")
	}
	// The index holds no secrets, but it does describe what exists and where,
	// so it stays owner-only.
	return os.WriteFile(path, data, 0o600)
}

// NewSecretID mints an id for a credential the launcher is about to save. The
// timestamp is passed in rather than read here so the result is testable.
func NewSecretID(connector, label string, now time.Time) string {
	id := "cnspec-ui-" + connector
	if label != "" {
		id += "-" + label
	}
	return id + "-" + now.UTC().Format("20060102T150405")
}
