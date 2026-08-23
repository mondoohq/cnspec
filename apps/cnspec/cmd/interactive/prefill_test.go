// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// Prefilling a guess the user has to notice and undo is worse than an empty
// field, so a value is only chosen when the answer is unambiguous.
func TestPreferredValueOnlyWhenUnambiguous(t *testing.T) {
	// Exactly one candidate is unambiguous whatever the source.
	if got, why := preferredValue(srcSSHHost, []string{"jump"}); got != "jump" || why != "only one" {
		t.Errorf("single host = %q/%q, want jump/only one", got, why)
	}
	// Several, with nothing to distinguish them, is not.
	if got, _ := preferredValue(srcSSHHost, []string{"jump", "prod-web"}); got != "" {
		t.Errorf("two hosts prefilled %q, want nothing", got)
	}
	// The conventional AWS name counts, even when the picker has labelled it.
	if got, why := preferredValue(srcAWSProfile, []string{"prod", "default  (123456789012)"}); got != "default  (123456789012)" || why != "default" {
		t.Errorf("aws = %q/%q, want the default profile", got, why)
	}
	// This machine's eight profiles include no "default", so nothing is chosen.
	if got, _ := preferredValue(srcAWSProfile, []string{"chris", "mondoo-dev", "private"}); got != "" {
		t.Errorf("profiles with no default prefilled %q, want nothing", got)
	}
	if got, _ := preferredValue(srcAWSProfile, nil); got != "" {
		t.Errorf("no values prefilled %q", got)
	}
}

// A source that reports after the user has started typing must not move the
// ground under them.
func TestPrefillNeverOverwritesTypedInput(t *testing.T) {
	m := selectEntry(t, newTestModel(), "ssh")
	m.focus = focusForm
	m.detail.form.SetCursor(0)
	m.detail.loadCursorField()
	m = typeString(m, "root@10.0.0.9")

	m.detail.applyPrefill(srcSSHHost, []string{"jump"}, m.focus == focusForm)
	if got := m.detail.form.Fields()[0].Value(); got != "root@10.0.0.9" {
		t.Fatalf("a late source overwrote typed input: %q", got)
	}
	if m.detail.form.Fields()[0].Prefilled() != "" {
		t.Error("a field the user typed must not be marked prefilled")
	}
}

// A prefilled field says why, so a value nobody typed is not a small mystery.
func TestPrefilledValueExplainsItself(t *testing.T) {
	m := selectEntry(t, sized(newTestModel(), 140, 40), "ssh")
	m.detail.applyPrefill(srcSSHHost, []string{"only-host"}, m.focus == focusForm)

	fd := m.detail.form.Fields()[0]
	if fd.Value() != "only-host" || fd.Prefilled() != "only one" {
		t.Fatalf("field = %q/%q, want it filled and explained", fd.Value(), fd.Prefilled())
	}
	if out := m.View(); !strings.Contains(out, "only one") {
		t.Error("the reason a field filled itself should be on screen")
	}
}

// The cheap sources run when a form opens so the target is already there; the
// ones that reach a cluster wait to be asked.
func TestCheapSourcesLoadOnOpenAndRemoteOnesDoNot(t *testing.T) {
	if deferredSource(srcSSHHost) || deferredSource(srcDockerContainer) {
		t.Error("file and local-daemon sources should not be deferred")
	}
	if !deferredSource(srcK8sNamespace) {
		t.Error("a source that queries a cluster should be deferred")
	}

	m := selectEntry(t, newTestModel(), "ssh")
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(Model)
	if m.focus != focusForm {
		t.Fatalf("expected the fields to open, got %d", m.focus)
	}
	if cmd == nil {
		t.Fatal("opening the fields should start the cheap pickers")
	}
}

// Prefilling must leave the command runnable, not merely populated.
func TestPrefillMakesTheCommandComplete(t *testing.T) {
	m := selectEntry(t, newTestModel(), "ssh")
	if m.readyToRun() {
		t.Fatal("ssh is not runnable before its target is known")
	}
	m.detail.applyPrefill(srcSSHHost, []string{"only-host"}, m.focus == focusForm)
	if !m.readyToRun() {
		t.Fatal("a prefilled target should leave the command runnable")
	}

	conn, _ := m.list.current()
	if got := m.commandPreview(conn); got != "cnspec scan ssh only-host" {
		t.Fatalf("command = %q", got)
	}
	if !strings.Contains(m.runButton(), "scan") {
		t.Fatalf("affordance should now offer to scan, got %q", m.runButton())
	}
}

// k8s asks what shape of thing is being scanned first, and then only what that
// shape needs. A cluster screen must not ask for a manifest path, and a
// manifest screen must not ask which cluster.
func TestK8sFormIsGuidedByWhatToScan(t *testing.T) {
	c := Connector{
		Provider: "k8s", Name: "k8s", Use: "k8s", Category: catContainer,
		Installed: true, MaxArgs: 1,
		Flags: []plugin.Flag{
			{Long: "context", Type: plugin.FlagType_String},
			{Long: "namespaces", Type: plugin.FlagType_String},
			{Long: "kubelogin", Type: plugin.FlagType_Bool},
		},
		Discovery: []string{"pods", "namespaces"},
	}
	f := newForm(c)

	labels := func() []string {
		var out []string
		for _, i := range f.VisibleIndices() {
			out = append(out, f.Fields()[i].Label)
		}
		return out
	}

	// Before anything is chosen, only the question itself is asked.
	if got := labels(); !containsString(got, "what to scan") {
		t.Fatalf("expected the shape question, got %v", got)
	}
	for _, hidden := range []string{"manifest path", "cluster", "namespaces (optional)"} {
		if containsString(labels(), hidden) {
			t.Errorf("%q should be hidden until a shape is chosen: %v", hidden, labels())
		}
	}

	fieldByLabel(t, f, "what to scan").SetValue("live cluster")
	if got := labels(); !containsString(got, "cluster") || !containsString(got, "namespaces (optional)") {
		t.Errorf("a live cluster should ask for a cluster and namespaces, got %v", got)
	}
	if containsString(labels(), "manifest path") {
		t.Errorf("a live cluster must not ask for a manifest path: %v", labels())
	}

	fieldByLabel(t, f, "what to scan").SetValue("manifest file")
	if got := labels(); !containsString(got, "manifest path") {
		t.Errorf("a manifest should ask for a path, got %v", got)
	}
	for _, hidden := range []string{"cluster", "namespaces (optional)"} {
		if containsString(labels(), hidden) {
			t.Errorf("a manifest must not ask for %q: %v", hidden, labels())
		}
	}
}

// A value left behind by an abandoned branch must not travel with the command.
func TestHiddenFieldsDoNotReachTheCommand(t *testing.T) {
	c := Connector{
		Provider: "k8s", Name: "k8s", Use: "k8s", Installed: true, MaxArgs: 1,
		Flags: []plugin.Flag{{Long: "context", Type: plugin.FlagType_String}},
	}
	f := newForm(c)
	fieldByLabel(t, f, "what to scan").SetValue("manifest file")
	fieldByLabel(t, f, "manifest path").SetValue("./deploy.yaml")
	// Set a cluster, then switch away from the branch that asked for it.
	fieldByLabel(t, f, "cluster").SetValue("some-cluster")

	got := strings.Join(f.Args(), " ")
	if strings.Contains(got, "some-cluster") {
		t.Fatalf("a hidden field reached the command: %q", got)
	}
	if got != "./deploy.yaml" {
		t.Fatalf("args = %q, want just the manifest path", got)
	}
}

// A hidden required field cannot block the scan, or choosing a manifest would
// be refused for want of a cluster.
func TestHiddenRequiredFieldDoesNotBlock(t *testing.T) {
	c := Connector{Provider: "k8s", Name: "k8s", Use: "k8s", Installed: true, MaxArgs: 1}
	f := newForm(c)
	fieldByLabel(t, f, "what to scan").SetValue("live cluster")
	if err := f.Validate(); err != nil {
		t.Fatalf("a live cluster should not need the manifest path: %v", err)
	}
}

// A single-letter shortcut in a pane full of text fields steals keystrokes:
// an "o" shortcut turned root@host into rot@host. Every letter must reach the
// field being edited.
func TestLettersReachTheFieldBeingEdited(t *testing.T) {
	m := selectEntry(t, newTestModel(), "ssh")
	m = key(m, tea.KeyTab)
	m = typeString(m, "root@options.example")

	if got := m.detail.form.Fields()[m.detail.form.Cursor()].Value(); got != "root@options.example" {
		t.Fatalf("typed value came out as %q", got)
	}
}

// The long tail of provider flags folds away behind one row, which is a focus
// stop rather than a shortcut.
func TestOptionsFoldAwayBehindAFocusStop(t *testing.T) {
	m := selectEntry(t, sized(newTestModel(), 120, 40), "ssh")
	m = key(m, tea.KeyTab)

	if !m.detail.hasMoreRow() {
		t.Fatal("ssh declares options, so there should be a row to reveal them")
	}
	for _, i := range m.detail.focusableFields() {
		if m.detail.form.Fields()[i].Section == sectionOptions {
			t.Fatalf("%q should be folded away", m.detail.form.Fields()[i].Label)
		}
	}
	if out := ansi.Strip(m.View()); !strings.Contains(out, "more options") {
		t.Errorf("the fold should be visible:\n%s", out)
	}

	// Tab to it and open it.
	for i := 0; i < 20 && !m.detail.onMore(); i++ {
		m = key(m, tea.KeyTab)
	}
	if !m.detail.onMore() {
		t.Fatal("tab should reach the reveal row")
	}
	m = key(m, tea.KeyEnter)
	if !m.detail.showOptions {
		t.Fatal("enter should reveal the options")
	}
	found := false
	for _, i := range m.detail.focusableFields() {
		if m.detail.form.Fields()[i].Section == sectionOptions {
			found = true
		}
	}
	if !found {
		t.Error("revealed options should be reachable")
	}
}

// A modal row must never wrap.
//
// The previous version of this test only checked lines against the terminal
// width, which the box never approaches, so it passed while every row inside
// the box wrapped by exactly the padding. This one measures the box itself: a
// wrapped row makes the box taller than the rows it is showing.
func TestModalRowsNeverWrap(t *testing.T) {
	long := []string{
		"arn:aws:eks:us-east-2:921877552404:cluster/patrick-container-escape-demo-azql-cluster",
		"default/api-mondoo-rosa1-49nm-p1-openshiftapps-com:6443/chris@mondoo.com",
		"gke_coding-agents-495409_us-central1_airlock",
		"short",
	}
	for _, size := range []struct{ w, h int }{{100, 30}, {80, 24}, {140, 40}} {
		m := selectEntry(t, sized(newTestModel(), size.w, size.h), "ssh")
		setSource(&m, srcSSHHost, long)
		m = key(m, tea.KeyTab)
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = nm.(Model)
		if !m.picker.modal.open {
			t.Fatalf("%dx%d: expected the picker to open", size.w, size.h)
		}

		lines := strings.Split(ansi.Strip(m.View()), "\n")
		if len(lines) != size.h {
			t.Fatalf("%dx%d: view is %d lines, want %d", size.w, size.h, len(lines), size.h)
		}

		// The box is every line carrying a border glyph.
		boxW, contentW := modalGeom(size.w)
		var box []string
		for _, ln := range lines {
			if strings.ContainsAny(ln, "│╭╰") {
				box = append(box, strings.TrimSpace(ln))
			}
		}
		if len(box) == 0 {
			t.Fatalf("%dx%d: no modal box rendered", size.w, size.h)
		}

		// Every box line is the same width, and that width is what the geometry
		// says. A wrapped row would not change this, so height is checked next.
		for i, ln := range box {
			if w := tui.Width(ln); w != boxW+2*modalBorder {
				t.Errorf("%dx%d: box line %d is %d wide, want %d", size.w, size.h, i, w, boxW+2*modalBorder)
			}
		}

		// 2 border + 2 padding + title + desc + blank + 4 rows + blank + help.
		wantHeight := 2 + 2 + 1 + 1 + 1 + len(long) + 1 + 1
		if len(box) != wantHeight {
			t.Errorf("%dx%d: box is %d lines, want %d — a row wrapped",
				size.w, size.h, len(box), wantHeight)
		}

		// Every value is on exactly one line, and the long ones keep their tail
		// so they can be told apart.
		joined := strings.Join(box, "\n")
		if !strings.Contains(joined, "cluster/patrick-container-escape-demo-azql-cluster") &&
			!strings.Contains(joined, "azql-cluster") {
			t.Errorf("%dx%d: the identifying end of the ARN was cut:\n%s", size.w, size.h, joined)
		}
		_ = contentW
	}
}

// Cutting the head of an over-long value keeps the part that identifies it.
func TestTruncateKeepsTheTail(t *testing.T) {
	arn := "arn:aws:eks:us-east-2:921877552404:cluster/patrick1"
	got := truncateTail(arn, 20)
	if tui.Width(got) != 20 {
		t.Fatalf("width = %d, want 20 (%q)", tui.Width(got), got)
	}
	if !strings.HasSuffix(got, "patrick1") {
		t.Fatalf("truncation dropped the cluster name: %q", got)
	}
	if !strings.HasPrefix(got, "…") {
		t.Fatalf("a cut value should say so: %q", got)
	}
	if short := truncateTail("chris-dev", 20); short != "chris-dev" {
		t.Fatalf("a value that fits should be untouched, got %q", short)
	}
}

// gcp is guided the same way k8s is: ask what is being scanned, then only what
// that answer needs.
func TestGCPFormIsGuidedByWhatToScan(t *testing.T) {
	c := Connector{
		Provider: "gcp", Name: "gcp", Use: "gcp", Category: catCloud,
		Installed: true, MaxArgs: 2,
		Flags: []plugin.Flag{
			{Long: "project-id", Type: plugin.FlagType_String},
			{Long: "zone", Type: plugin.FlagType_String},
			{Long: "repository", Type: plugin.FlagType_String},
			{Long: "credentials-path", Type: plugin.FlagType_String, Desc: "The path to the service account credentials"},
		},
	}
	f := newForm(c)
	labels := func() []string {
		var out []string
		for _, i := range f.VisibleIndices() {
			out = append(out, f.Fields()[i].Label)
		}
		return out
	}

	fieldByLabel(t, f, "what to scan").SetValue("project")
	fieldByLabel(t, f, "id").SetValue("attack-surface-scanner")
	if got := strings.Join(f.Args(), " "); got != "project attack-surface-scanner" {
		t.Fatalf("args = %q, want the sub-command the connector documents", got)
	}
	for _, hidden := range []string{"project", "zone", "repository"} {
		for _, l := range labels() {
			if l == hidden && hidden != "project" {
				t.Errorf("a project scan must not ask for %q", hidden)
			}
		}
	}

	// An instance is addressed by project and zone.
	fieldByLabel(t, f, "what to scan").SetValue("compute instance")
	if got := labels(); !containsString(got, "project") || !containsString(got, "zone") {
		t.Errorf("an instance should ask for project and zone, got %v", got)
	}
	fieldByLabel(t, f, "id").SetValue("web-1")
	fieldByLabel(t, f, "project").SetValue("attack-surface-scanner")
	fieldByLabel(t, f, "zone").SetValue("us-central1-c")
	got := strings.Join(f.Args(), " ")
	if got != "instance web-1 --project-id attack-surface-scanner --zone us-central1-c" {
		t.Fatalf("args = %q", got)
	}

	// A registry is addressed by repository, not by zone.
	fieldByLabel(t, f, "what to scan").SetValue("container registry")
	if got := labels(); !containsString(got, "repository") || containsString(got, "zone") {
		t.Errorf("a registry should ask for a repository and not a zone, got %v", got)
	}
}

// The active gcloud project is prefilled, the way the current kube context is.
func TestGCPPrefillsTheActiveProject(t *testing.T) {
	value, why := preferredValue(srcGCPProject, []string{"other-project", gcpActiveProject()})
	if gcpActiveProject() == "" {
		t.Skip("no gcloud configuration on this machine")
	}
	if value != gcpActiveProject() || why != "active" {
		t.Fatalf("prefill = %q/%q, want the active project", value, why)
	}
	// Two projects with no active one is ambiguous, so nothing is chosen.
	if got, _ := preferredValue(srcGCPProject, []string{"a-project", "b-project"}); got != "" {
		t.Errorf("prefilled %q where the answer is ambiguous", got)
	}
}

// The cheap local read and the live list are merged, so the configured project
// never disappears because the network call was slow or refused.
func TestLiveValuesMergeWithLocalOnes(t *testing.T) {
	c := Connector{
		Provider: "gcp", Name: "gcp", Use: "gcp", Category: catCloud,
		Installed: true, MaxArgs: 2,
		Flags: []plugin.Flag{{Long: "project-id", Type: plugin.FlagType_String}},
	}
	m := NewModel([]Connector{c})
	m.syncSelection()
	fieldByLabel(t, m.detail.form, "what to scan").SetValue("project")
	resolveSources(&m.detail.form) // the selector decides which picker the id uses

	id := fieldByLabel(t, m.detail.form, "id")
	if id.Source() != srcGCPProject || id.LiveSource != srcGCPProjectAll {
		t.Fatalf("id field sources = %q/%q, want the local read plus the live list", id.Source(), id.LiveSource)
	}

	setSource(&m, srcGCPProject, []string{"attack-surface-scanner"})
	m.picker.fill(&m.detail.form)
	if got := fieldByLabel(t, m.detail.form, "id").Options; len(got) != 1 {
		t.Fatalf("before the live call the picker should hold the configured project, got %v", got)
	}

	setSource(&m, srcGCPProjectAll, []string{"mondoo-demo", "attack-surface-scanner"})
	m.picker.fill(&m.detail.form)
	got := fieldByLabel(t, m.detail.form, "id").Options
	if len(got) != 2 || !containsString(got, "attack-surface-scanner") || !containsString(got, "mondoo-demo") {
		t.Fatalf("merged options = %v, want both without duplicates", got)
	}
}

// Opening the picker starts the live call; entering the form does not.
func TestLiveListWaitsUntilThePickerOpens(t *testing.T) {
	if !deferredSource(srcGCPProjectAll) {
		t.Fatal("the live project list must not run on entering a form")
	}
	if deferredSource(srcGCPProject) {
		t.Fatal("the configured project is a file read and should load immediately")
	}
}

// A live list belongs to the selector that asked for it. Left attached across a
// change of kind, an organization id would be offered a list of projects, and
// the picker would sit there fetching them.
func TestLiveSourceFollowsTheSelector(t *testing.T) {
	c := Connector{
		Provider: "gcp", Name: "gcp", Use: "gcp", Category: catCloud,
		Installed: true, MaxArgs: 2,
	}
	f := newForm(c)
	kind := fieldByLabel(t, f, "what to scan")

	for _, tc := range []struct{ kind, source, live string }{
		{"project", srcGCPProject, srcGCPProjectAll},
		{"container registry", srcGCPProject, srcGCPProjectAll},
		{"organization", "", ""},
		{"folder", "", ""},
		{"compute instance", "", ""},
	} {
		kind.SetValue(tc.kind)
		resolveSources(&f)
		id := fieldByLabel(t, f, "id")
		if id.Source() != tc.source || id.LiveSource != tc.live {
			t.Errorf("kind %q: sources = %q/%q, want %q/%q",
				tc.kind, id.Source(), id.LiveSource, tc.source, tc.live)
		}
	}
}

// A field with nothing to offer and nowhere to get anything is a text field.
// Opening an empty box over it hides the only thing that works: typing.
func TestNoPickerWhenThereIsNothingToPick(t *testing.T) {
	m := selectEntry(t, sized(newTestModel(), 120, 40), "aws")
	m = key(m, tea.KeyTab)
	// Find a plain text field with no source.
	for i, fd := range m.detail.form.Fields() {
		if fd.Kind == fieldChoice && fd.Source() == "" && fd.LiveSource == "" && len(fd.Options) == 0 {
			m.detail.form.SetCursor(i)
			nm, _ := m.openModal()
			if nm.(Model).picker.modal.open {
				t.Fatalf("%q has nothing to offer but opened a picker", fd.Label)
			}
		}
	}
}

// A wait says what it is waiting for, and names the tool, so the user knows
// which credentials matter and what to run if it goes wrong.
func TestWaitNamesTheToolBeingAsked(t *testing.T) {
	for src, want := range map[string]string{
		srcGCPProjectAll: "gcloud",
		srcK8sNamespace:  "kubectl",
		srcKubeContext:   "kubectl",
		srcDockerImage:   "docker",
	} {
		if got := activityFor(src); !strings.Contains(got, want) {
			t.Errorf("activity for %q = %q, want it to name %q", src, got, want)
		}
	}

	m := selectEntry(t, sized(newTestModel(), 120, 40), "ssh")
	m = key(m, tea.KeyTab)
	fd := m.detail.form.Fields()[m.detail.form.Cursor()]
	m.picker.begin(sourceKeyFor(m.detail.form, fd.Source()))
	if got := m.picker.waitingFor(m.detail.form, fd); !strings.Contains(got, "ssh") {
		t.Fatalf("waitingFor = %q, want it to name what is being read", got)
	}
	m.picker.modal = modalState{open: true, field: m.detail.form.Cursor()}
	if out := ansi.Strip(m.View()); !strings.Contains(out, "reading ~/.ssh/config") {
		t.Errorf("the picker should say what it is waiting for:\n%s", out)
	}
}
