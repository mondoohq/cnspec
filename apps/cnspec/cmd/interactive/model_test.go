// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
)

// sampleCatalog is a small hermetic catalog so model tests don't depend on the
// installed provider set.
func sampleCatalog() []Connector {
	// ssh and aws carry the metadata a real installed provider would, so the
	// model tests exercise actual forms rather than empty ones.
	ssh := sshConnector()
	ssh.Short = "a remote system via SSH"
	aws := awsConnector()
	aws.Short = "an AWS account"

	return []Connector{
		{Provider: "os", Name: "local", Use: "local", Short: "your local system", Category: catHosts, Installed: true},
		ssh,
		{Provider: "os", Name: "docker", Use: "docker", Short: "a running Docker container", Category: catContainer},
		aws,
		{Provider: "k8s", Name: "k8s", Use: "k8s", Short: "a Kubernetes cluster", Category: catContainer},
		{Provider: "mongo", Name: "mongo", Use: "mongo [host]", Short: "a MongoDB server", Category: catDatabase},
	}
}

// setSource seeds a picker's cached values the way a completed load would.
func setSource(m *Model, source string, values []string) {
	m.picker.answer(sourceValuesMsg{source: source, key: sourceKeyFor(m.detail.form, source), values: values})
	m.picker.fill(&m.detail.form)
}

// setSourceErr records why a picker came back empty, the way a load that could
// not reach its cluster would. It goes through the same answer path a real one
// does, so a test cannot leave the four facts about a key in a combination the
// launcher would never produce.
func setSourceErr(m *Model, source string, err error) {
	m.picker.answer(sourceValuesMsg{source: source, key: sourceKeyFor(m.detail.form, source), err: err})
}

// newTestModel builds the launcher as a machine that is connected to Mondoo
// Platform, because otherwise every assertion about an assembled command line
// would depend on whether the developer running the suite happens to be logged
// in -- the same command gains --incognito on a machine with no credentials.
// Tests that care about the other states set upstream themselves.
func newTestModel() Model {
	m := NewModel(sampleCatalog())
	m.upstream = upstreamState{configured: true, scope: "test-space"}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return nm.(Model)
}

func typeString(m Model, s string) Model {
	for _, r := range s {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = nm.(Model)
	}
	return m
}

func key(m Model, t tea.KeyType) Model {
	nm, cmd := m.Update(tea.KeyMsg{Type: t})
	return settle(nm.(Model), cmd)
}

// settle delivers the one message a test has to see for itself, the way the
// program's own loop would.
//
// The scan plan is assembled off the event loop -- see prepareLaunchCmd, which
// is what stops a locked keychain freezing the screen -- so pressing the button
// no longer finishes the launch inside Update. It is gated on the launch state
// rather than run for every command, because most of what this model returns
// is a timer: textinput.Blink and spinner.Tick both sleep when called, and a
// test that ran them would pay half a second per keystroke.
func settle(m Model, cmd tea.Cmd) Model {
	if !m.launching.preparing || cmd == nil {
		return m
	}
	// launchCmd comes back from this, and is deliberately dropped: tea.Exec
	// only wraps the child, so nothing is spawned by building it.
	nm, _ := m.Update(cmd())
	return nm.(Model)
}

// selectEntry moves the cursor onto the named entry.
func selectEntry(t *testing.T, m Model, name string) Model {
	t.Helper()
	for i := range m.list.selectable {
		m.list.cursor = i
		m.list.ensureVisible(m.listH())
		m.syncSelection()
		if e, ok := m.list.current(); ok && e.Name == name {
			return m
		}
	}
	t.Fatalf("entry %q not found", name)
	return m
}

func TestFilterNarrowsResults(t *testing.T) {
	m := typeString(newTestModel(), "aws")
	if len(m.list.filtered) != 1 || m.list.filtered[0].Name != "aws" {
		t.Fatalf("expected only aws to match, got %d results", len(m.list.filtered))
	}

	m2 := typeString(newTestModel(), "database")
	if len(m2.list.filtered) != 1 || m2.list.filtered[0].Name != "mongo" {
		t.Fatalf("expected category search to find mongo, got %d results", len(m2.list.filtered))
	}
}

// The list opens populated and ready: no landing screen, no action list, and
// the first connector's fields already built.
func TestOpensReadyToLaunch(t *testing.T) {
	m := newTestModel()
	if m.focus != focusList {
		t.Fatalf("expected list focus on open, got %d", m.focus)
	}
	e, ok := m.list.current()
	if !ok || e.Name != "local" {
		t.Fatalf("expected local selected first, got %q ok=%v", e.Name, ok)
	}
	// local needs nothing, so opening the launcher and pressing enter is a
	// complete interaction.
	if !m.readyToRun() {
		t.Fatal("expected local to be runnable with no input")
	}
}

// Choosing a connector never starts a scan. Configuring a target and running
// one are separate acts; enter from the list only opens the configuration.
func TestEnterFromTheListNeverScans(t *testing.T) {
	for _, name := range []string{"aws", "k8s", "local"} {
		m := selectEntry(t, newTestModel(), name)
		m = key(m, tea.KeyEnter)
		if m.lastRun != "" {
			t.Errorf("%s: enter from the list launched %q", name, m.lastRun)
		}
		if m.focus != focusForm {
			t.Errorf("%s: expected the fields to open, got focus %d", name, m.focus)
		}
	}
}

// tabToScan walks the fields to the scan button.
func tabToScan(m Model) Model {
	for i := 0; i < 40 && !m.detail.onButton(); i++ {
		m = key(m, tea.KeyTab)
	}
	return m
}

// The scan button is the only thing that scans.
func TestOnlyTheButtonScans(t *testing.T) {
	m := selectEntry(t, newTestModel(), "aws")
	m = key(m, tea.KeyTab) // into the fields
	if m.detail.onButton() {
		t.Fatal("the fields should be entered on a field, not the button")
	}

	m = tabToScan(m)
	if !m.detail.onButton() {
		t.Fatal("tab should reach the scan button")
	}
	m = key(m, tea.KeyEnter)
	if m.lastRun != "cnspec scan aws" {
		t.Fatalf("expected the button to launch, got %q lastErr=%q", m.lastRun, m.lastErr)
	}
}

// A required field still blocks, and says which one.
func TestButtonRefusesWhileSomethingIsRequired(t *testing.T) {
	m := selectEntry(t, newTestModel(), "ssh")
	m = key(m, tea.KeyTab)
	m = tabToScan(m)
	m = key(m, tea.KeyEnter)

	if m.lastRun != "" {
		t.Fatalf("expected no launch, got %q", m.lastRun)
	}
	if !strings.Contains(m.lastErr, "user@host") {
		t.Fatalf("expected the error to name the missing field, got %q", m.lastErr)
	}
}

// Tab walks the fields and then the button; shift+tab walks back and off the
// top returns to the list.
func TestFocusWalksFieldsThenButton(t *testing.T) {
	m := selectEntry(t, newTestModel(), "ssh")
	m = key(m, tea.KeyTab)
	if m.focus != focusForm || m.detail.onButton() {
		t.Fatalf("expected the first field, got focus=%d button=%v", m.focus, m.detail.onButton())
	}
	first := m.detail.form.Cursor()

	m = key(m, tea.KeyTab)
	if m.detail.form.Cursor() == first && !m.detail.onButton() {
		t.Fatal("tab should move to the next field")
	}

	m = tabToScan(m)
	m = key(m, tea.KeyTab) // already at the end, stays put
	if !m.detail.onButton() {
		t.Fatal("tab past the button should stay on the button")
	}

	// Back out the way we came.
	for i := 0; i < 40 && m.focus == focusForm; i++ {
		m = key(m, tea.KeyShiftTab)
	}
	if m.focus != focusList {
		t.Fatalf("shift+tab off the top should return to the list, got %d", m.focus)
	}
}

// The keys are in exactly one place in the detail pane, and carrying a position
// out of one pane must not light up two rows in the next.
//
// This is the bug the three-valued spot ended. moreFocused and scanFocused were
// two bools, and leaving the pane from the reveal row left moreFocused set. Open
// a connector whose every field is a folded option -- `local`, which has two --
// and enterForm set scanFocused as well, because there was no reachable field
// for the keys to land on. Both were then true, the renderer drew the selection
// dot on the reveal row *and* on the scan button, and the two disagreed about
// what a key meant: moveFocus read "more", enter read "button".
//
// Reproduced from the keyboard on the shipped catalog before the fix:
// ssh, enter, down x4 (onto the reveal row), esc, esc, down x2, enter.
func TestOnlyOneRowOfThePaneHasTheKeys(t *testing.T) {
	m := selectEntry(t, newTestModel(), "ssh")
	m = key(m, tea.KeyTab)
	for i := 0; i < 40 && !m.detail.onMore(); i++ {
		m = key(m, tea.KeyTab)
	}
	if !m.detail.onMore() {
		t.Fatal("tab never reached the row that reveals the folded options")
	}

	// Out of the pane, onto a connector with nothing reachable to fill in, and
	// back in. The position from the other pane must not survive as a second
	// highlighted row.
	m = key(m, tea.KeyEsc)
	m = selectEntry(t, m, "local")
	if len(m.detail.focusableFields()) != 0 {
		t.Skip("local now has a reachable field; the premise no longer holds")
	}
	m = key(m, tea.KeyTab)

	if !m.detail.onButton() {
		t.Fatalf("a pane with nothing to fill in should open on the button, got spot=%d", m.detail.spot)
	}
	if m.detail.onMore() {
		t.Fatal("the reveal row kept the keys it was given in another pane")
	}
}

// esc clears the filter before it quits, so a typo does not drop the session.
func TestEscClearsFilterFirst(t *testing.T) {
	m := typeString(newTestModel(), "aws")
	m = key(m, tea.KeyEsc)
	if m.list.search.Value() != "" {
		t.Fatalf("expected esc to clear the filter, got %q", m.list.search.Value())
	}
	if len(m.list.filtered) != len(m.list.entries) {
		t.Fatalf("expected the full list back, got %d of %d", len(m.list.filtered), len(m.list.entries))
	}
}

func TestActionFilteringForConnector(t *testing.T) {
	// aws does not support sbom; ensure it is not offered.
	for _, a := range ActionsFor("aws") {
		if a.Name == "sbom" {
			t.Fatal("sbom should not be offered for aws")
		}
	}
	names := map[string]bool{}
	for _, a := range ActionsFor("local") {
		names[a.Name] = true
	}
	for _, want := range []string{"scan", "shell", "run", "discover", "vuln", "sbom", "aibom"} {
		if !names[want] {
			t.Fatalf("expected action %q for local", want)
		}
	}
}

func TestTokenizeQuotes(t *testing.T) {
	got := tokenize(`-c "aws.regions.length"`)
	if len(got) != 2 || got[0] != "-c" || got[1] != "aws.regions.length" {
		t.Fatalf("unexpected tokenize result: %#v", got)
	}
}

func TestRequiresArg(t *testing.T) {
	cases := []struct {
		use, name string
		want      bool
	}{
		{"ssh user@host", "ssh", true},
		{"ansible PATH", "ansible", true},
		{"mongo [host]", "mongo", false},
		{"sbom [flags]", "sbom", false},
		{"aws", "aws", false},
	}
	for _, c := range cases {
		if got := (Connector{Name: c.name, Use: c.use}).RequiresArg(); got != c.want {
			t.Errorf("RequiresArg(%q) = %v, want %v", c.use, got, c.want)
		}
	}
}

// withTestSource registers a source for one test and takes it out again, so
// the contract tests that walk the whole registry never see it.
func withTestSource(t *testing.T, s Source) {
	t.Helper()
	if _, taken := sourceByID(s.ID); taken {
		t.Fatalf("%s is already registered; pick an id no real source uses", s.ID)
	}
	register(s)
	t.Cleanup(func() { delete(registry, s.ID) })
}

// A load has to be registered under the key its answer will arrive with.
//
// It was not. pendingSourceCmds registered m.loading[sourceKeyFor(m.detail.form, src)] -- a
// key built from the parameters the source declared it needs -- and then
// started the load with nil parameters, whose answer carries a key built from
// nil. The delete in the sourceValuesMsg handler therefore matched nothing and
// the field waited for an answer it had already been given. Only the one
// CostRemote source had Needs when this was written, and that one skips this
// path, which is why nothing shipped span; the enumerated and discovery
// sources put dependent fields straight onto it.
func TestAPendingLoadIsAnsweredUnderTheKeyItRegistered(t *testing.T) {
	const id = "test.needs-a-cluster"
	withTestSource(t, Source{
		ID:       id,
		Class:    ClassPostConnection,
		Cost:     CostInstant,
		Activity: "reading the test fixture",
		Tool:     "the test fixture",
		Needs:    []string{"context"},
		Fetch:    func([]string) ([]string, error) { return []string{"kube-system"}, nil },
	})

	m := newTestModel()
	m.detail.form.SetFields([]field{
		valued(tuiform.Decl{Label: "context", Flag: "context", Kind: fieldChoice}, "prod-eu"),
		sourced(tuiform.Decl{Label: "namespace", Flag: "namespace", Kind: fieldChoice}, id),
	})

	cmds := m.picker.pendingCmds(m.detail.form)
	if len(cmds) != 1 {
		t.Fatalf("expected one pending load, got %d", len(cmds))
	}
	waits := m.picker.inFlight()
	if len(waits) != 1 {
		t.Fatalf("expected one registered wait, got %v", waits)
	}
	registered := waits[0]
	// The parameters are the point: a key with no cluster in it would be a
	// different cluster's namespace list.
	if !strings.Contains(registered, "prod-eu") {
		t.Errorf("the wait was registered without the cluster it is for: %q", registered)
	}

	msg, ok := cmds[0]().(sourceValuesMsg)
	if !ok {
		t.Fatalf("expected a sourceValuesMsg, got %T", cmds[0]())
	}
	if msg.key != registered {
		t.Fatalf("the answer arrived under %q but the wait was registered as %q;\n"+
			"the spinner can never be cleared", msg.key, registered)
	}

	nm, _ := m.Update(msg)
	if left := nm.(Model).picker.inFlight(); len(left) != 0 {
		t.Errorf("the wait outlived its answer: %v", left)
	}
}

// The keychain write must not happen on the event loop.
//
// A locked keychain does not fail, it asks: macOS raises an authentication
// dialog and the call does not return until it is answered. Doing that inside
// Update froze the whole TUI behind a dialog the launcher was not drawing.
func TestTheKeychainWriteIsNotOnTheEventLoop(t *testing.T) {
	release := make(chan struct{})
	orig := storeCredentialFn
	storeCredentialFn = func(id string, cred *vault.Credential) error {
		<-release
		return errors.New("keyring unavailable")
	}
	defer func() { storeCredentialFn = orig }()

	m := selectEntry(t, newTestModel(), "ssh")
	m.detail.form.Fields()[fieldIndex(t, m.detail.form, "user@host")].SetValue("chris@10.0.0.4")
	// identity-file has no --ask-* partner, so this is the inventory route --
	// the one that puts a credential in the keychain. A password would be
	// prompted for by the child and never reach one.
	secret := &m.detail.form.Fields()[fieldIndex(t, m.detail.form, "identity-file")]
	secret.Secret, secret.Reference = true, false
	secret.SetValue("<PLACEHOLDER-not-a-real-secret>")
	m.focus = focusForm
	m.detail.spot = spotButton

	type result struct {
		m   Model
		cmd tea.Cmd
	}
	done := make(chan result, 1)
	go func() {
		nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		done <- result{nm.(Model), cmd}
	}()

	var got result
	select {
	case got = <-done:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("Update did not return while the keychain write was outstanding")
	}
	if !got.m.launching.preparing {
		t.Error("the button has nothing to say while the scan is being prepared")
	}
	if got.m.lastRun != "" {
		t.Errorf("the launch finished inside Update: %q", got.m.lastRun)
	}

	// And the answer still arrives, with the existing fallback intact: the
	// scan runs, the secret is off the command line, and the user is told the
	// guarantee is weaker.
	close(release)
	msg, ok := got.cmd().(launchPreparedMsg)
	if !ok {
		t.Fatalf("expected a prepared plan, got %T", got.cmd())
	}
	if msg.err != nil {
		t.Fatalf("a keychain failure must not stop the scan: %v", msg.err)
	}
	if msg.plan.cleanup != nil {
		defer msg.plan.cleanup()
	}
	if !strings.Contains(msg.plan.warn, "keychain") {
		t.Errorf("falling back to a file must warn, got %q", msg.plan.warn)
	}
	if strings.Contains(strings.Join(msg.plan.args, " "), "<PLACEHOLDER-not-a-real-secret>") {
		t.Errorf("secret reached argv: %v", msg.plan.args)
	}

	nm, _ := got.m.Update(msg)
	after := nm.(Model)
	if after.launching.preparing {
		t.Error("the button is still preparing after the plan arrived")
	}
	if after.lastRun == "" || after.lastWarn == "" {
		t.Errorf("the launch was not reported: run=%q warn=%q", after.lastRun, after.lastWarn)
	}
}

// A picker owns what it starts, so closing one stops it. Abandoning the result
// and leaving `gcloud projects list` or `cnspec discover` running to
// completion is not the same thing.
func TestClosingAPickerStopsWhatItStarted(t *testing.T) {
	const id = "test.blocks-until-cancelled"
	started := make(chan struct{})
	withTestSource(t, Source{
		ID:       id,
		Class:    ClassPostConnection,
		Cost:     CostRemote,
		Activity: "asking the test fixture for nothing",
		Tool:     "the test fixture",
		FetchCtx: func(ctx context.Context, _ []string) ([]string, error) {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})

	// The load itself: cancelled means cancelled, and says so rather than
	// reporting the killed child's own complaint as a failure of the cluster.
	ctx, cancel := context.WithCancel(context.Background())
	answers := make(chan tea.Msg, 1)
	go func() { answers <- loadSourceCmd(ctx, id, nil)() }()
	<-started
	cancel()

	var msg sourceValuesMsg
	select {
	case got := <-answers:
		msg, _ = got.(sourceValuesMsg)
	case <-time.After(5 * time.Second):
		t.Fatal("the fetch outlived its context")
	}
	if !msg.cancelled {
		t.Errorf("a cancelled load reported itself as an answer: %+v", msg)
	}

	// And the model's half: opening the picker records how to stop the load,
	// closing it runs that and clears the wait with it.
	m := newTestModel()
	m.detail.form.SetFields([]field{
		sourced(tuiform.Decl{Label: "project", Flag: "project", Kind: fieldChoice}, id),
	})
	if cmd := m.openPickerCmd(m.detail.form.Fields()[0]); cmd == nil {
		t.Fatal("opening the picker started nothing")
	}
	if len(m.picker.cancellable()) != 1 || len(m.picker.inFlight()) != 1 {
		t.Fatalf("the picker's load was not tracked: cancels=%d loading=%d",
			len(m.picker.cancellable()), len(m.picker.inFlight()))
	}
	m.picker.close()
	if left := m.picker.cancellable(); len(left) != 0 {
		t.Errorf("closing the picker left %d loads running", len(left))
	}
	if left := m.picker.inFlight(); len(left) != 0 {
		t.Errorf("closing the picker left the field waiting: %v", left)
	}
}

// An explicit DOCKER_HOST silently overrides the context the launcher sets, so
// the child is cleared of it -- but only when the user actually chose a
// context, and never for `default`, which *is* the DOCKER_HOST resolution.
func TestDockerHostCannotOverrideTheChosenContext(t *testing.T) {
	env := func(t *testing.T, value string) []string {
		t.Helper()
		fd := sourced(tuiform.Decl{Label: "context", Kind: fieldChoice}, srcDockerContext)
		fd.SetValue(value)
		r := launchRequest{form: tuiform.New("docker", []field{fd})}
		got, cleanup, err := r.environment()
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	chosen := env(t, "colima")
	if !containsString(chosen, dockerContextEnv+"=colima") {
		t.Fatalf("the chosen context did not reach the child: %v", chosen)
	}
	if !containsString(chosen, dockerHostEnv+"=") {
		t.Errorf("DOCKER_HOST still overrides the chosen context: %v", chosen)
	}

	// `default` is not a host of its own; it is the DOCKER_HOST-and-socket
	// resolution, so clearing the variable there would retarget the scan.
	fallback := env(t, dockerDefaultContext)
	for _, e := range fallback {
		if strings.HasPrefix(e, dockerHostEnv+"=") {
			t.Errorf("the default context cleared DOCKER_HOST: %v", fallback)
		}
	}
}

// Quitting removes what the launcher wrote. The generated inventory holds a
// plaintext credential whenever the keychain was unavailable, and it used to
// be removed on exactly one event: the command it fed finishing.
func TestQuittingRemovesTheGeneratedInventory(t *testing.T) {
	inv := t.TempDir()
	path := filepath.Join(inv, "inventory.yml")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newTestModel()
	m.launching.cleanup = trackTemp(func() { _ = os.RemoveAll(inv) })

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c did not quit")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("quitting left the generated inventory behind: %v", err)
	}
	if nm.(Model).launching.cleanup != nil {
		t.Error("the launcher still holds a cleanup it has already run")
	}

	// And the same from the list's own esc, which is the ordinary way out.
	inv2 := t.TempDir()
	m2 := newTestModel()
	m2.launching.cleanup = trackTemp(func() { _ = os.RemoveAll(inv2) })
	if _, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyEsc}); cmd == nil {
		t.Fatal("esc did not quit an empty search")
	}
	if _, err := os.Stat(inv2); !os.IsNotExist(err) {
		t.Errorf("quitting with esc left %s behind", inv2)
	}
}

// fieldIndex finds a field by label.
func fieldIndex(t *testing.T, f form, label string) int {
	t.Helper()
	for i, fd := range f.Fields() {
		if fd.Label == label {
			return i
		}
	}
	t.Fatalf("no field labelled %q", label)
	return -1
}
