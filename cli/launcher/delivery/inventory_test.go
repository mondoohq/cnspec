// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package delivery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
)

// sshForm is a host-shaped form with a password in it, built here rather than
// through the launcher's connector catalog: what this package promises is about
// the file it writes, and it should be checkable without a screen. The
// launcher's own tests cover the other half -- that its curated ssh form
// produces exactly this shape.
func sshForm() tuiform.Form {
	target := tuiform.NewField(tuiform.Decl{
		Label: "user@host", Pos: 1, Kind: tuiform.KindText,
		Section: tuiform.SectionTarget,
	})
	target.SetValue("chris@10.0.0.4")

	password := tuiform.NewField(tuiform.Decl{
		Label: "password", Flag: "password", Kind: tuiform.KindText,
		Secret: true, Section: tuiform.SectionCredential,
	})
	password.SetValue("must-never-reach-argv")

	return tuiform.New("ssh", []tuiform.Field{target, password})
}

// sshAsset is what the os provider's own ParseCLI answers for that form: a
// connection with a host and a typed credential carrying the user.
//
// It is written out rather than fetched, because a unit test that started a
// plugin subprocess would be testing the plugin. That the real providers
// actually answer this way is asserted separately, over whatever is installed;
// see TestEveryCredentialFieldRoundTrips in the launcher.
func sshAsset() *inventory.Asset {
	return &inventory.Asset{
		Connections: []*inventory.Config{{
			Type: "ssh",
			Host: "10.0.0.4",
			Credentials: []*vault.Credential{{
				Type: vault.CredentialType_password, User: "chris", Password: "must-never-reach-argv",
			}},
		}},
	}
}

// The whole point of the vault route: the secret goes to the OS keychain and
// the file carries an id, so the password is in neither the process table nor
// anything the launcher wrote.
func TestInventoryReferencesAVaultSecret(t *testing.T) {
	f, asset := sshForm(), sshAsset()
	saved := Keychainable(Locate(f, asset))
	if saved == nil {
		t.Fatal("the password did not land anywhere the keychain can hold")
	}
	inv := InventoryFor("ssh", asset, saved, "cnspec-ui-ssh-chris-20260818T120000")

	creds := inv.Spec.Assets[0].Connections[0].Credentials
	if len(creds) != 1 {
		t.Fatalf("want exactly one credential reference, got %d", len(creds))
	}
	ref := creds[0]
	if ref.SecretId != "cnspec-ui-ssh-chris-20260818T120000" {
		t.Errorf("secret id = %q", ref.SecretId)
	}
	// The loader rejects a reference that also declares a type, and silently
	// discards inline material when an id is present.
	if ref.Type != vault.CredentialType_undefined {
		t.Errorf("a reference must not declare a type, got %v", ref.Type)
	}
	if ref.Password != "" || len(ref.Secret) != 0 {
		t.Error("a reference must carry no inline secret")
	}
	if inv.Spec.Vault == nil || inv.Spec.Vault.Type != vault.VaultType_KeyRing {
		t.Fatalf("want the keyring vault named in the inventory, got %+v", inv.Spec.Vault)
	}

	data, err := inv.ToYAML()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "must-never-reach-argv") {
		t.Fatalf("the secret was written into the inventory despite the vault:\n%s", data)
	}
}

// What the keychain holds is the credential the *provider* built, type, user
// and all -- not a password with no user assembled here.
//
// This is the difference the ParseCLI route makes and it is worth an assertion
// of its own: artifactory needs a bearer credential and spends a password as
// the legacy API key against the wrong header, azure and ms365 want pkcs12, and
// okta wants private_key. Flattening every one of them into a password is what
// the code this replaced did.
func TestTheKeychainHoldsTheProvidersOwnCredential(t *testing.T) {
	asset := &inventory.Asset{Connections: []*inventory.Config{{
		Type: "artifactory",
		Credentials: []*vault.Credential{{
			Type: vault.CredentialType_bearer, User: "token", Password: "must-never-reach-argv",
		}},
	}}}
	saved := Keychainable(Locate(sshForm(), asset))
	if saved == nil {
		t.Fatal("the value did not land in a credential")
	}
	cred := saved.Credential()
	if cred.Type != vault.CredentialType_bearer || cred.User != "token" {
		t.Fatalf("the launcher would save %v/%q rather than what the provider built",
			cred.Type, cred.User)
	}
}

// The fallback route -- a keychain that was unavailable -- writes the
// credential to disk, so the file it writes has to be one only this user can
// read, and it has to be gone afterwards.
func TestWriteInventoryIsPrivateAndCleansUp(t *testing.T) {
	path, cleanup, err := WriteInventory(InventoryFor("ssh", sshAsset(), nil, ""))
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("inventory mode = %o, want 600", perm)
	}
	// The directory too: on some systems the file name alone leaks.
	dir, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if perm := dir.Mode().Perm(); perm != 0o700 {
		t.Errorf("inventory directory mode = %o, want 700", perm)
	}

	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("inventory survived cleanup: %v", err)
	}
}

func TestNewSecretIDIsStableAndScoped(t *testing.T) {
	at := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	got := NewSecretID("ssh", "chris", at)
	if got != "cnspec-ui-ssh-chris-20260818T120000" {
		t.Fatalf("id = %q", got)
	}
	if NewSecretID("ssh", "", at) != "cnspec-ui-ssh-20260818T120000" {
		t.Fatalf("id with no label = %q", NewSecretID("ssh", "", at))
	}
}

// Everything the launcher writes is removed on every exit it can observe, not
// only on the tidy one.
//
// The generated inventory holds a plaintext credential whenever the OS
// keychain was unavailable, and cleanupLaunch used to run on exactly one
// event: the command it fed reporting that it had finished. A quit before the
// scan, a signal, or a panic each left that file in the system temp directory.
func TestWhatTheLauncherWritesIsRemovedOnEveryExit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.yml")
	if err := os.WriteFile(path, []byte("password: must-never-reach-argv\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cleanup := TrackTemp(func() { _ = os.RemoveAll(dir) })
	CleanupTempFiles()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("the file survived the exit hook: %v", err)
	}

	// Both the launch path and the exit path legitimately hold a cleanup, so
	// running one after the other must be free rather than a double removal of
	// whatever now sits at that path.
	cleanup()
	cleanup()
	CleanupTempFiles()
}

// A cleanup that has already run is not run again by the exit hook, which is
// what keeps a second launch's directory from being removed underneath it.
func TestACompletedCleanupIsForgotten(t *testing.T) {
	runs := 0
	cleanup := TrackTemp(func() { runs++ })
	cleanup()
	CleanupTempFiles()
	if runs != 1 {
		t.Fatalf("the cleanup ran %d times, want once", runs)
	}
}

// The generated inventory registers itself, so nothing has to remember to.
func TestTheGeneratedInventoryIsTracked(t *testing.T) {
	path, cleanup, err := WriteInventory(InventoryFor("ssh", sshAsset(), nil, ""))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the inventory was not written: %v", err)
	}
	// Not through the returned cleanup: through the exit hook, which is the
	// path a quit or a signal takes.
	CleanupTempFiles()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the generated inventory outlived the exit hook: %v", err)
	}
}

// A provider can build two credentials at once -- a password and a private key
// path -- and only one of them is the keychain's.
//
// The keychain holds a value. A key path is not a value to store, it is a
// reference the provider resolves itself, so it has to stay in the file
// whatever the keychain did. It did not: the reference used to replace the
// credential list wholesale, so the key path survived when the keychain was
// unavailable and disappeared when it worked -- the broken behaviour lived on
// the path that succeeds. Both directions are checked here, because the bug was
// that they disagreed.
func TestAKeyPathSurvivesTheKeychain(t *testing.T) {
	const key = "/home/chris/.ssh/id_ed25519"

	build := func(secretID string) []*vault.Credential {
		t.Helper()
		asset := &inventory.Asset{Connections: []*inventory.Config{{
			Type: "ssh", Host: "10.0.0.4",
			Credentials: []*vault.Credential{
				{Type: vault.CredentialType_password, User: "chris", Password: "must-never-reach-argv"},
				{Type: vault.CredentialType_private_key, User: "chris", PrivateKeyPath: key},
			},
		}}}
		saved := Keychainable(Locate(sshForm(), asset))
		if saved == nil {
			t.Fatal("the password did not land anywhere the keychain can hold")
		}
		inv := InventoryFor("ssh", asset, saved, secretID)
		return inv.Spec.Assets[0].Connections[0].Credentials
	}

	for _, tc := range []struct {
		name     string
		secretID string
	}{
		{name: "keychain unavailable", secretID: ""},
		{name: "keychain holds the password", secretID: "cnspec-ui-ssh-chris-20260818T120000"},
	} {
		creds := build(tc.secretID)

		var sawKeyPath, sawPassword, sawReference bool
		for _, cred := range creds {
			switch {
			case cred.SecretId != "":
				sawReference = true
			case cred.Type == vault.CredentialType_private_key:
				sawKeyPath = cred.PrivateKeyPath == key
			case cred.Type == vault.CredentialType_password:
				sawPassword = true
			}
		}

		if !sawKeyPath {
			t.Errorf("%s: the private key path is not in the inventory: %+v", tc.name, creds)
		}
		if wantReference := tc.secretID != ""; sawReference != wantReference {
			t.Errorf("%s: keychain reference present = %v, want %v", tc.name, sawReference, wantReference)
		}
		// The password travels one way or the other, never both: a reference
		// beside the value it references would put the secret back in the file.
		if sawPassword == (tc.secretID != "") {
			t.Errorf("%s: the password is written into the file = %v, and referenced = %v",
				tc.name, sawPassword, sawReference)
		}
	}
}
