// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"os"
	"strings"
	"testing"

	deliverypkg "go.mondoo.com/cnspec/cli/launcher/delivery"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
)

// Writing the inventory lives in cli/launcher/delivery, and the properties of
// the file itself -- 0600 in a 0700 directory, removed on every exit, a vault
// reference carrying no inline secret -- are tested there against an asset
// written by hand.
//
// What is tested here is the join, end to end and against a real provider: that
// the launcher's *curated* ssh form, handed to the os provider's own ParseCLI,
// produces an asset that cnspec's inventory loader accepts and that the
// keychain can protect. A field renamed or re-sectioned in a spec breaks this
// and nothing in the delivery package would notice.
//
// ssh is the right connector to spend a subprocess on. It is the one whose
// inventory shape cnspec's own documentation and testdata demonstrate, so a
// disagreement here is a disagreement with something written down.

func sshFormWithPassword(t *testing.T) (Connector, form) {
	t.Helper()
	c := sshConnector()
	f := newForm(c)
	fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
	fieldByLabel(t, f, "password").SetValue("<PLACEHOLDER-not-a-real-secret>")
	fieldByLabel(t, f, "sudo").SetOn(true)
	return c, f
}

// sshAssetFromProvider asks the os provider what the curated ssh form means.
//
// It skips rather than fails when the provider is not installed, and says so:
// CI points PROVIDERS_PATH at an empty directory, so a test in this shape that
// stayed silent would be green having checked nothing.
func sshAssetFromProvider(t *testing.T, f form) (Connector, *inventory.Asset) {
	t.Helper()
	c := sshConnector()
	if !installedProviders()[c.Provider] {
		t.Skipf("the %s provider is not installed here, so the ssh join was not checked", c.Provider)
	}
	req := deliverypkg.RequestFor(f, c.Flags)
	asset, err := deliverypkg.Parser.ParseCLI(c.Provider, c.Name, req.Args, req.Flags)
	if err != nil {
		t.Fatalf("the os provider refused the curated ssh form: %v", err)
	}
	return c, asset
}

// The generated inventory must load through the same code path `cnspec scan`
// uses, or the launcher is writing files only it can read.
func TestGeneratedInventoryRoundTrips(t *testing.T) {
	_, f := sshFormWithPassword(t)
	c, asset := sshAssetFromProvider(t, f)

	data, err := deliverypkg.InventoryFor(c.Name, asset, nil, "").ToYAML()
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := inventory.InventoryFromYAML(data)
	if err != nil {
		t.Fatalf("cnspec cannot read the inventory we generated: %v\n%s", err, data)
	}
	if err := loaded.PreProcess(); err != nil {
		t.Fatalf("PreProcess rejected it: %v\n%s", err, data)
	}
	if err := loaded.Validate(); err != nil {
		t.Fatalf("Validate rejected it: %v\n%s", err, data)
	}
}

// The curated ssh form maps onto the shape the os provider reads: a host, a
// user carried by the credential rather than by the target string, and sudo.
func TestInventoryShapesTheSSHAsset(t *testing.T) {
	_, f := sshFormWithPassword(t)
	_, asset := sshAssetFromProvider(t, f)
	conn := asset.Connections[0]

	if conn.Type != "ssh" {
		t.Errorf("connection type = %q, want ssh", conn.Type)
	}
	// The deprecated `backend` field triggers a fallback path and a warning.
	if conn.Backend != 0 {
		t.Errorf("backend should be left unset, got %v", conn.Backend)
	}
	if conn.Host != "10.0.0.4" {
		t.Errorf("host = %q, want 10.0.0.4", conn.Host)
	}
	if conn.Sudo == nil || !conn.Sudo.Active {
		t.Error("sudo should be active")
	}
	// Two credentials, and that is the provider's decision rather than the
	// launcher's: a password for what was typed and an ssh_agent fallback for
	// the same user, so a wrong password does not cost you the agent. The
	// launcher's own builder produced one or the other and never both.
	t.Logf("the os provider built %d credentials: %s",
		len(conn.Credentials), describeCreds(conn.Credentials))

	var password *vault.Credential
	for _, cred := range conn.Credentials {
		if cred.Type == vault.CredentialType_password {
			password = cred
		}
	}
	if password == nil {
		t.Fatalf("no password credential among %s", describeCreds(conn.Credentials))
	}
	if password.User != "chris" {
		t.Errorf("credential user = %q, want chris", password.User)
	}
	// Which field of the credential holds it is the provider's business -- ssh
	// puts it in Secret rather than Password -- and Locate is what knows how to
	// look, which is why the launcher never has to.
	placed := deliverypkg.Locate(f, asset)
	if len(placed) != 1 || placed[0].Placement != deliverypkg.PlacedCredential {
		t.Fatalf("the typed password was located as %+v, want a credential", placed)
	}
}

func describeCreds(creds []*vault.Credential) string {
	var out []string
	for _, c := range creds {
		out = append(out, c.Type.String()+"/user="+c.User)
	}
	return strings.Join(out, ", ")
}

// With no credential in the form, a host connector still carries its user so
// ssh can fall back to the agent for that account.
//
// This used to be something the launcher arranged, in a branch of its own
// inventory builder. It is the provider's own behaviour, which is why it is
// worth asserting here rather than there: if the os provider ever stopped doing
// it, the launcher would silently stop offering agent authentication.
func TestInventoryFallsBackToSSHAgent(t *testing.T) {
	c := sshConnector()
	f := newForm(c)
	fieldByLabel(t, f, "user@host").SetValue("deploy@host.example")

	_, asset := sshAssetFromProvider(t, f)
	creds := asset.Connections[0].Credentials
	if len(creds) != 1 || creds[0].Type != vault.CredentialType_ssh_agent || creds[0].User != "deploy" {
		t.Fatalf("want a single ssh_agent credential for deploy, got %+v", creds)
	}
}

// A vault-backed launch references the secret by id, and the password the user
// typed into the launcher's own form must not reach the file.
func TestInventoryReferencesAVaultSecret(t *testing.T) {
	_, f := sshFormWithPassword(t)
	c, asset := sshAssetFromProvider(t, f)

	saved := deliverypkg.Keychainable(deliverypkg.Locate(f, asset))
	if saved == nil {
		t.Fatal("the os provider kept the password nowhere the keychain can hold")
	}
	inv := deliverypkg.InventoryFor(c.Name, asset, saved, "cnspec-ui-ssh-chris-20260818T120000")

	creds := inv.Spec.Assets[0].Connections[0].Credentials
	refs := 0
	for _, cred := range creds {
		if cred.SecretId != "" {
			refs++
			if cred.SecretId != "cnspec-ui-ssh-chris-20260818T120000" {
				t.Errorf("secret id = %q", cred.SecretId)
			}
		}
	}
	// Exactly one, whatever else the provider built alongside it: the reference
	// stands in for the credential that was saved and for no other.
	if refs != 1 {
		t.Fatalf("want exactly one credential reference, got %d of %s",
			refs, describeCreds(creds))
	}
	if inv.Spec.Vault == nil || inv.Spec.Vault.Type != vault.VaultType_KeyRing {
		t.Fatalf("want the keyring vault named in the inventory, got %+v", inv.Spec.Vault)
	}

	// The whole point: the password must not be in the file.
	data, err := inv.ToYAML()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "<PLACEHOLDER-not-a-real-secret>") {
		t.Fatalf("the secret was written into the inventory despite the vault:\n%s", data)
	}
}

// The generated inventory is removed on every exit the launcher can observe,
// including the ones that are not a finished scan.
func TestTheGeneratedInventoryIsTracked(t *testing.T) {
	_, f := sshFormWithPassword(t)
	c, asset := sshAssetFromProvider(t, f)

	path, cleanup, err := writeInventory(deliverypkg.InventoryFor(c.Name, asset, nil, ""))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the inventory was not written: %v", err)
	}
	// Not through the returned cleanup: through the exit hook, which is the
	// path a quit or a signal takes.
	cleanupTempFiles()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the generated inventory outlived the exit hook: %v", err)
	}
}
