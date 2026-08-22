// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
	deliverypkg "go.mondoo.com/cnspec/cli/launcher/delivery"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/mql/cli/inventoryloader"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
)

// An exported inventory is only worth anything if `cnspec scan --inventory-file`
// can read it, so nothing here asserts that a file is non-empty yaml. Every
// write goes back in through inventoryloader.Parse, which is the function
// scan.go calls (apps/cnspec/cmd/scan.go, via ParseOrUse) -- the same reading,
// the same PreProcess, the same failure if a credential reference is malformed.

// stubKeychain replaces the OS keychain write and the index it records to.
//
// Without it every test here would raise a real authentication dialog on the
// developer's machine and then leave a real credential in their login keyring.
// The returned map is what was "saved", keyed by the id the inventory will
// reference.
func stubKeychain(t *testing.T) map[string]*vault.Credential {
	t.Helper()
	saved := map[string]*vault.Credential{}
	prevStore, prevRecord := storeCredentialFn, recordSaved
	t.Cleanup(func() { storeCredentialFn, recordSaved = prevStore, prevRecord })

	storeCredentialFn = func(id string, cred *vault.Credential) error {
		saved[id] = cred
		return nil
	}
	recordSaved = func(savedEntry) error { return nil }
	return saved
}

// loadExported reads a written file the way `cnspec scan --inventory-file` does.
//
// The path travels through viper because that is the only way in: Parse reads
// the flag out of the global config rather than taking an argument, so a test
// that wants the real loader has to set it the way cobra would.
func loadExported(t *testing.T, path string) *inventory.Inventory {
	t.Helper()
	prev := viper.GetString("inventory-file")
	viper.Set("inventory-file", path)
	t.Cleanup(func() { viper.Set("inventory-file", prev) })

	inv, err := inventoryloader.Parse()
	if err != nil {
		data, _ := os.ReadFile(path)
		t.Fatalf("cnspec cannot read the inventory we exported: %v\n%s", err, data)
	}
	if inv == nil || len(inv.Spec.GetAssets()) == 0 {
		t.Fatalf("the exported inventory loaded with no assets in it")
	}
	return inv
}

// exportSSHForm is the filled ssh form the export tests start from: a target
// and a password, which is the shape that puts a credential in the keychain.
func exportSSHForm(t *testing.T) (Connector, form) {
	t.Helper()
	c := sshConnector()
	f := newForm(c)
	fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
	fieldByLabel(t, f, "password").SetValue(sentinel)
	return c, f
}

// exportTo runs the export the way the box does and returns the path written.
func exportTo(t *testing.T, c Connector, f form, path string) {
	t.Helper()
	r := launchRequest{form: f}
	parsed, err := r.parseForm(c)
	if err != nil {
		t.Fatalf("the provider refused the form: %v", err)
	}
	if err := r.exportInventory(c, parsed, path); err != nil {
		t.Fatalf("the export failed: %v", err)
	}
}

// The claim the whole feature rests on: what the box writes is a file the scan
// command reads.
func TestTheExportedInventoryLoadsThroughTheScanLoader(t *testing.T) {
	stubKeychain(t)
	withParser(t, &fakeParser{secretFlag: "password"})
	c, f := exportSSHForm(t)

	path := filepath.Join(t.TempDir(), "ssh.inventory.yml")
	exportTo(t, c, f, path)

	inv := loadExported(t, path)
	asset := inv.Spec.Assets[0]
	if len(asset.Connections) == 0 {
		t.Fatalf("the exported asset has no connection: %+v", asset)
	}
	if got := asset.Connections[0].Type; got != c.Name {
		t.Errorf("connection type = %q, want %q", got, c.Name)
	}
	// A loaded inventory that names the keyring is what makes the reference
	// resolvable at scan time; without it the id points at nothing.
	if inv.Spec.Vault == nil || inv.Spec.Vault.Type != vault.VaultType_KeyRing {
		t.Fatalf("the exported inventory does not name the OS keychain: %+v", inv.Spec.Vault)
	}
}

// The secret is in the keychain, and therefore not in the file.
//
// This is the assertion behind the sentence the box shows. If it ever fails,
// the box is telling the user their password stayed in the keychain while the
// file they just put in a repository holds it.
func TestTheExportedFileHoldsAReferenceAndNotTheSecret(t *testing.T) {
	keychain := stubKeychain(t)
	withParser(t, &fakeParser{secretFlag: "password"})
	c, f := exportSSHForm(t)

	path := filepath.Join(t.TempDir(), "ssh.inventory.yml")
	exportTo(t, c, f, path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), sentinel) {
		t.Fatalf("the exported file carries the secret:\n%s", data)
	}
	if len(keychain) != 1 {
		t.Fatalf("the keychain holds %d credentials, want the one the file references", len(keychain))
	}

	var id string
	for k := range keychain {
		id = k
	}
	inv := loadExported(t, path)
	refs := 0
	for _, conn := range inv.Spec.Assets[0].Connections {
		for _, cred := range conn.Credentials {
			if cred.SecretId == id {
				refs++
			}
		}
	}
	if refs != 1 {
		t.Fatalf("the loaded inventory references the saved credential %d times, want once", refs)
	}
}

// The exported file is the user's, so it is written where they said, once, and
// readable only by them.
func TestTheExportedFileIsPrivateAndNeverOverwritten(t *testing.T) {
	stubKeychain(t)
	withParser(t, &fakeParser{secretFlag: "password"})
	c, f := exportSSHForm(t)

	path := filepath.Join(t.TempDir(), "ssh.inventory.yml")
	exportTo(t, c, f, path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("the exported inventory is %v, want 0600", perm)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A second export to the same path must refuse rather than replace: the
	// launcher cannot tell a collision from an intention, and only one of the
	// two readings can be undone.
	r := launchRequest{form: f}
	parsed, err := r.parseForm(c)
	if err != nil {
		t.Fatal(err)
	}
	err = r.exportInventory(c, parsed, path)
	if err == nil {
		t.Fatal("exporting over an existing file was allowed")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("the refusal does not say what happened: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("the refused export still changed the file that was there")
	}
}

// A keychain that will not hold the credential refuses the export, where the
// same failure only warns a scan.
//
// The difference is the file. A scan falls back to a 0600 inventory that is
// deleted when it finishes; an export would fall back to a file at a path the
// user chose, holding their password, immediately after a box that told them
// the secret stays in the keychain.
func TestExportRefusesWhenTheKeychainWillNotHoldTheSecret(t *testing.T) {
	prev := storeCredentialFn
	t.Cleanup(func() { storeCredentialFn = prev })
	storeCredentialFn = func(string, *vault.Credential) error {
		return errors.New("keyring unavailable")
	}
	withParser(t, &fakeParser{secretFlag: "password"})
	c, f := exportSSHForm(t)

	path := filepath.Join(t.TempDir(), "ssh.inventory.yml")
	r := launchRequest{form: f}
	parsed, err := r.parseForm(c)
	if err != nil {
		t.Fatal(err)
	}
	err = r.exportInventory(c, parsed, path)
	if err == nil {
		t.Fatal("the export was written with the secret in it after the keychain failed")
	}
	if !strings.Contains(err.Error(), "keychain") || !strings.Contains(err.Error(), "plain text") {
		t.Errorf("the refusal does not explain itself: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a file was written anyway: %v", err)
	}

	// And the same failure still lets a scan run, which is the behaviour this
	// must not have changed.
	plan, planErr := r.plan(c, scanAction())
	if plan.cleanup != nil {
		defer plan.cleanup()
	}
	if planErr != nil {
		t.Fatalf("a keychain failure must not stop a scan: %v", planErr)
	}
	if plan.warn == "" {
		t.Error("the scan's fallback to a file stopped warning")
	}
}

// The four connectors whose provider keeps the key in conn.Options get a file
// with the secret in it -- and a box that said so first.
//
// Both halves are asserted together on purpose. The warning is only honest
// because the secret really is there, and the write is only defensible because
// the warning really is shown.
func TestAPlaintextProviderIsDisclosedAndThenExportedAsDisclosed(t *testing.T) {
	stubKeychain(t)
	// No secretFlag: every flag becomes a connection option, which is what
	// openai, ollama, huggingface and claude do with their key.
	withParser(t, &fakeParser{})
	withAmbientEnv(t, nil)

	c, f := formFor(t, "openai")
	fieldByLabel(t, f, "API key").SetValue(aiKey)

	r := launchRequest{form: f}
	parsed, err := r.parseForm(c)
	if err != nil {
		t.Fatalf("the provider refused the form: %v", err)
	}
	plain := parsed.plaintextFlags()
	if len(plain) != 1 || plain[0] != "token" {
		t.Fatalf("the key was located as %v, want the --token option", plain)
	}

	// What the user reads before pressing enter.
	note := exportPlaintextNote(c.Name, plain)
	for _, want := range []string{"openai", "--token", "plain text", "version control"} {
		if !strings.Contains(note, want) {
			t.Errorf("the disclosure does not mention %q: %q", want, note)
		}
	}

	path := filepath.Join(t.TempDir(), "openai.inventory.yml")
	if err := r.exportInventory(c, parsed, path); err != nil {
		t.Fatalf("the export failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), aiKey) {
		t.Fatalf("the key is not in the file, so the disclosure is wrong about it:\n%s", data)
	}
	// Disclosed is not the same as careless: the file is still owner-only, and
	// it still has to load.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("a file holding a key in the clear is %v, want 0600", perm)
	}
	loadExported(t, path)
}

// The box says the one thing the file's own contents cannot: that it depends on
// this machine's keychain, as this user.
func TestTheExportBoxNamesTheKeychainDependency(t *testing.T) {
	stubKeychain(t)
	withParser(t, &fakeParser{secretFlag: "password"})

	m := openedExportBox(t, "ssh", func(f form) {
		fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
		fieldByLabel(t, f, "password").SetValue(sentinel)
	})

	out := ansi.Strip(m.View())
	for _, want := range []string{"OS keychain", "this machine", "this user", "CI runner"} {
		if !strings.Contains(out, want) {
			t.Errorf("the export box never says %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, sentinel) {
		t.Errorf("the export box rendered the secret:\n%s", out)
	}
}

// And for a provider that cannot use the keychain it names the provider and the
// flag, on screen, before the write.
func TestTheExportBoxNamesThePlaintextProviderAndFlag(t *testing.T) {
	stubKeychain(t)
	withParser(t, &fakeParser{})
	withAmbientEnv(t, nil)

	m := openedExportBox(t, "openai", func(f form) {
		fieldByLabel(t, f, "API key").SetValue(aiKey)
	})

	out := ansi.Strip(m.View())
	for _, want := range []string{"openai", "--token", "plain text"} {
		if !strings.Contains(out, want) {
			t.Errorf("the export box never says %q:\n%s", want, out)
		}
	}
	// The key hint says what enter agrees to, not just that it writes.
	if !strings.Contains(out, "secret in plain text") {
		t.Errorf("the confirm key does not say what it agrees to:\n%s", out)
	}
	if strings.Contains(out, aiKey) {
		t.Errorf("the export box rendered the key:\n%s", out)
	}
}

// openedExportBox drives the launcher to an open, answered export box for one
// connector, through the keys a user would press.
func openedExportBox(t *testing.T, connector string, fill func(form)) Model {
	t.Helper()
	snap, ok := snapshotByName(t)[connector]
	if !ok {
		t.Fatalf("%s is not in %s", connector, connectorSnapshotPath)
	}
	c := snap.connector()

	m := NewModel([]Connector{c})
	m.upstream = upstreamState{configured: true, scope: "test-space"}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = nm.(Model)
	fill(m.detail.form)

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = nm.(Model)
	if !m.export.open || !m.export.asking {
		t.Fatalf("ctrl+e did not open the export box: %+v", m.export)
	}
	nm, _ = m.Update(runBatch(t, cmd, exportParsedMsg{}))
	m = nm.(Model)
	if !m.export.ready {
		t.Fatalf("the export box never got an answer: %v", m.export.err)
	}
	return m
}

// runBatch runs a batched command and returns the one message of the same type
// as want. The launcher batches its animation clock into everything, and the
// tick is not what a test is waiting for.
func runBatch(t *testing.T, cmd tea.Cmd, want tea.Msg) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command to run")
	}
	msgs := []tea.Msg{cmd()}
	if batch, ok := msgs[0].(tea.BatchMsg); ok {
		msgs = nil
		for _, c := range batch {
			if c != nil {
				msgs = append(msgs, c())
			}
		}
	}
	for _, msg := range msgs {
		if reflectSameType(msg, want) {
			return msg
		}
	}
	t.Fatalf("no %T came back from the command", want)
	return nil
}

func reflectSameType(a, b tea.Msg) bool {
	switch a.(type) {
	case exportParsedMsg:
		_, ok := b.(exportParsedMsg)
		return ok
	case exportWroteMsg:
		_, ok := b.(exportWroteMsg)
		return ok
	}
	return false
}

// The whole flow, key by key, ending in a file on disk and a footer that hands
// over the command that reads it.
func TestTheExportFlowWritesTheFileAndSaysHowToUseIt(t *testing.T) {
	stubKeychain(t)
	withParser(t, &fakeParser{secretFlag: "password"})

	dir := t.TempDir()
	m := openedExportBox(t, "ssh", func(f form) {
		fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
		fieldByLabel(t, f, "password").SetValue(sentinel)
	})
	if m.export.name != "ssh.inventory.yml" {
		t.Errorf("the box opened on %q, want the connector's own name", m.export.name)
	}

	// The name is typed over, the way a user names one target among several.
	for range len("ssh.inventory.yml") {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = nm.(Model)
	}
	if m.export.name != "" {
		t.Fatalf("backspace left %q behind", m.export.name)
	}
	m = typeString(m, filepath.Join(dir, "prod-web.yml"))

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if !m.export.writing {
		t.Fatal("enter did not start the write")
	}
	nm, _ = m.Update(runBatch(t, cmd, exportWroteMsg{}))
	m = nm.(Model)

	if m.export.open {
		t.Error("the box stayed open after a successful write")
	}
	if m.lastErr != "" {
		t.Fatalf("the export reported an error: %s", m.lastErr)
	}
	path := filepath.Join(dir, "prod-web.yml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("nothing was written to the path that was typed: %v", err)
	}
	if !strings.Contains(m.lastNote, "cnspec scan --inventory-file") {
		t.Errorf("the footer does not say how to use the file: %q", m.lastNote)
	}
	if !strings.Contains(ansi.Strip(m.View()), "cnspec scan --inventory-file") {
		t.Error("the footer notice never reached the screen")
	}
	loadExported(t, path)
}

// esc leaves without writing, and the answer still in flight lands nowhere.
func TestEscapeClosesTheExportBoxWithoutWriting(t *testing.T) {
	stubKeychain(t)
	withParser(t, &fakeParser{secretFlag: "password"})

	m := openedExportBox(t, "ssh", func(f form) {
		fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
		fieldByLabel(t, f, "password").SetValue(sentinel)
	})
	seq := m.export.seq

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if m.export.open {
		t.Fatal("esc did not close the export box")
	}
	// The provider's answer to the box that was just closed must not reopen it.
	nm, _ = m.Update(exportParsedMsg{seq: seq, parsed: parsedForm{}})
	if m := nm.(Model); m.export.open || m.export.ready {
		t.Fatalf("a late answer revived the closed box: %+v", m.export)
	}
}

// An incomplete form does not export. It puts the keys on the field that is
// missing instead, which is what pressing the scan button does with the same
// form.
func TestExportingAnIncompleteFormPointsAtTheMissingField(t *testing.T) {
	m := newTestModel()
	m = selectEntry(t, m, "ssh")

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = nm.(Model)
	if m.export.open {
		t.Fatal("an incomplete form opened the export box")
	}
	if m.lastErr == "" {
		t.Error("nothing said why the export did not happen")
	}
	if m.focus != focusForm {
		t.Error("the keys were not moved to the field that is missing")
	}
}

// The footer offers the key only for a form that could be launched, the way it
// offers ^o only once a report exists.
func TestTheExportHintAppearsWithACompleteTarget(t *testing.T) {
	m := newTestModel()
	m = selectEntry(t, m, "ssh")
	if strings.Contains(ansi.Strip(m.View()), "^e") {
		t.Error("the export key is offered for a form that cannot be launched")
	}

	fieldByLabel(t, m.detail.form, "user@host").SetValue("chris@10.0.0.4")
	if !strings.Contains(ansi.Strip(m.View()), "^e") {
		t.Errorf("the export key is not offered for a complete target:\n%s", ansi.Strip(m.View()))
	}
}

// Where the file goes is decided here rather than by a shell, so the two
// spellings a shell would have handled are handled.
func TestTheExportPathIsResolvedBeforeAnythingIsWritten(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory here")
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, typed, want, wantErr string
	}{
		{name: "relative", typed: "ssh.inventory.yml", want: filepath.Join(wd, "ssh.inventory.yml")},
		{name: "tilde", typed: "~/scans/prod.yml", want: filepath.Join(home, "scans", "prod.yml")},
		{name: "absolute", typed: "/tmp/prod.yml", want: "/tmp/prod.yml"},
		{name: "empty", typed: "   ", wantErr: "name"},
		{name: "bare tilde", typed: "~", wantErr: "directory"},
		{name: "a directory", typed: wd, wantErr: "is a directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := exportState{name: tc.typed}.path()
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("%q resolved to %q, want a refusal", tc.typed, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("the refusal for %q does not mention %q: %v", tc.typed, tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("%q was refused: %v", tc.typed, err)
			}
			if got != tc.want {
				t.Errorf("%q resolved to %q, want %q", tc.typed, got, tc.want)
			}
		})
	}
}

// A file name is typed, and a name is not bytes.
//
// The picker's filter had the same handling and the same bug: one backspace cut
// one byte off the end, so a multi-byte character left half of itself behind.
// In a filter that renders as a replacement glyph; in a path it is a file name
// with an invalid byte in it.
func TestBackspaceRemovesACharacterRatherThanAByte(t *testing.T) {
	x := exportState{name: "scan-münchen.yml"}
	x.name = trimLastRune(x.name)
	x.name = trimLastRune(x.name)
	x.name = trimLastRune(x.name)
	x.name = trimLastRune(x.name)
	if x.name != "scan-münchen" {
		t.Fatalf("four backspaces left %q", x.name)
	}
	for range len([]rune(x.name)) {
		x.name = trimLastRune(x.name)
	}
	if x.name != "" {
		t.Errorf("backspacing every character left %q", x.name)
	}

	s := modalState{filter: "prüfung"}
	s.backspace()
	if s.filter != "prüfun" {
		t.Errorf("the picker filter lost a byte rather than a character: %q", s.filter)
	}
}

// A file name can hold a space, so the box has to be able to take one.
//
// bubbletea reports a lone space as tea.KeySpace rather than tea.KeyRunes, with
// the rune still attached, so a handler that matches only KeyRunes drops every
// space without a sign. What that produced was "myscans.yml" from a user who
// had typed "my scans.yml", which is a file written somewhere they did not ask
// for under a name they did not choose.
func TestASpaceCanBeTypedIntoTheFileName(t *testing.T) {
	stubKeychain(t)
	withParser(t, &fakeParser{secretFlag: "password"})

	m := openedExportBox(t, "ssh", func(f form) {
		fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
		fieldByLabel(t, f, "password").SetValue(sentinel)
	})
	m.export.name = ""

	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'m', 'y'}},
		{Type: tea.KeySpace, Runes: []rune{' '}},
		{Type: tea.KeyRunes, Runes: []rune{'s', 'c', 'a', 'n', 's'}},
	} {
		nm, _ := m.Update(msg)
		m = nm.(Model)
	}
	if m.export.name != "my scans" {
		t.Fatalf("the typed name came out as %q", m.export.name)
	}
}

// While a modal owns the body, a click must not reach the screen behind it.
//
// The keys have always been routed to whichever box is up. The mouse was not,
// so the zones computeLayout registers for a list and a detail pane that are
// not being drawn stayed live underneath: clicking where the scan button would
// have been started a scan from behind an open box, and the wheel moved the
// connector selection -- which rebuilds the form the box is describing.
func TestAClickDoesNotReachTheScreenBehindAModal(t *testing.T) {
	stubKeychain(t)
	withParser(t, &fakeParser{secretFlag: "password"})

	m := openedExportBox(t, "ssh", func(f form) {
		fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
		fieldByLabel(t, f, "password").SetValue(sentinel)
	})

	// Where the scan button is drawn when no box is over it.
	var run tui.Rect
	for _, z := range computeLayout(m).Zones {
		if z.Kind == tui.ZoneRun {
			run = z.Rect
		}
	}
	if run.W == 0 {
		t.Fatal("the layout registers no scan button to click")
	}

	nm, cmd := m.Update(tea.MouseMsg{
		X: run.X, Y: run.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	got := nm.(Model)
	if got.launching.preparing || cmd != nil {
		t.Error("a click behind the export box started a scan")
	}
	if !got.export.open {
		t.Error("the click closed the box it was supposed to be blocked by")
	}

	// And the wheel leaves the selection where the box found it.
	before := got.list.cursor
	nm, _ = got.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	if after := nm.(Model).list.cursor; after != before {
		t.Errorf("the wheel moved the selection under the box: %d to %d", before, after)
	}
}

// The gate, over the real providers: every connector the launcher can offer is
// exported and read back through the scan loader.
//
// This is the equivalent of the report viewer's TestEveryOfferedFormatWrites,
// and it is here for the same reason: a format that cannot be written, or a
// shape the loader rejects, is only discovered by writing one and reading it
// back. What it would catch is a provider whose asset does not survive a
// round trip through yaml -- an option key yaml cannot represent, a credential
// the loader refuses -- for a connector nobody exported by hand.
//
// The counts are logged and the test skips when it checked nothing. CI points
// PROVIDERS_PATH at an empty directory, so a total asserted here would pass
// vacuously there, and the vacuous pass is the dangerous half.
func TestEveryConnectorExportsSomethingTheLoaderAccepts(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a provider plugin per connector")
	}
	stubKeychain(t)

	// The real parser, not the package-wide stand-in TestMain installs: the
	// question here is whether what the *providers* build survives being
	// written and read back.
	prev := parseCLI
	t.Cleanup(func() { parseCLI = prev })
	parseCLI = deliverypkg.Parser.ParseCLI

	dir := t.TempDir()
	installed := installedProviders()

	checked, skipped, refused := 0, 0, 0
	withKeychain := 0
	// Named rather than counted. The package comment in export.go says openai,
	// ollama, huggingface and claude are the connectors whose secret cannot go
	// to the keychain, and the only honest way to keep that sentence true is to
	// print the set every run rather than to assert the four.
	var plaintext []string
	for _, c := range BuildCatalog() {
		if !c.DeclaresMetadata() || !installed[c.Provider] {
			skipped++
			continue
		}
		f := fillOneCredential(c, firstSecretIdentity(c))
		r := launchRequest{form: f}
		parsed, err := r.parseForm(c)
		if err != nil {
			// Either the provider refused the probe's settings or it kept a
			// secret nowhere -- both are facts about the filling rather than
			// about the export, and both are what a user sees as a refusal.
			refused++
			t.Logf("%s: not exportable from this probe (%v)", c.Name, err)
			continue
		}

		path := filepath.Join(dir, c.Name+".inventory.yml")
		if err := r.exportInventory(c, parsed, path); err != nil {
			t.Errorf("%s: the export failed: %v", c.Name, err)
			continue
		}
		checked++
		if plain := parsed.plaintextFlags(); len(plain) > 0 {
			plaintext = append(plaintext, c.Name+" "+flagList(plain))
		} else if len(parsed.placed) > 0 {
			withKeychain++
		}
		loadExported(t, path)
	}

	sort.Strings(plaintext)
	t.Logf("exported and re-read %d connectors: %d put a credential in the keychain, "+
		"%d were refused by the provider, %d catalog entries skipped for having no "+
		"metadata installed here.\n%d write a secret in plain text and say so: %s",
		checked, withKeychain, refused, skipped, len(plaintext), strings.Join(plaintext, ", "))
	// Skipped rather than passed when most of the catalog was unavailable.
	//
	// "checked == 0" is not a strict enough gate here, and the difference is
	// not theoretical: with PROVIDERS_PATH pointed at an empty directory this
	// still finds one connector to export and would otherwise report a green
	// pass for a claim about a hundred. The claim is about the provider set, so
	// a run that did not have the provider set has not tested it.
	if checked <= skipped {
		t.Skipf("only %d of %d catalog entries had a provider installed here, so this did "+
			"not check the provider set", checked, checked+skipped)
	}
}

// firstSecretIdentity names the credential field the probe fills, or nothing
// for a connector that has none.
func firstSecretIdentity(c Connector) string {
	f := newForm(c)
	for i := range f.Fields() {
		fd := f.Fields()[i]
		if fd.Secret && !fd.Reference && fd.Flag != "" && f.Visible(i) {
			return fd.Identity()
		}
	}
	return ""
}
