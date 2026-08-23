// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cockroachdb/errors"
	deliverypkg "go.mondoo.com/cnspec/cli/launcher/delivery"
	"go.mondoo.com/cnspec/cli/tui"
)

// A scan configured here is already an inventory: the connector's own ParseCLI
// builds the asset, the launcher writes it to a 0600 file, points a child at it
// and deletes it. Exporting keeps that file instead of deleting it, at a path
// the user names, which is the whole feature -- an exported inventory is a
// valid `cnspec scan --inventory-file` input, so a cron entry or a CI job runs
// the identical scan without anybody retyping the flags.
//
// # What the box has to say, and why it is not the same as the scan's warning
//
// The launcher puts a credential in the OS keychain and leaves a reference to
// it in the inventory (see delivery/parsecli.go: vault.VaultType_KeyRing, with
// no vault picker, because a TUI is not where a CI runner is configured). That
// is invisible and harmless for a file that lives for the length of one scan,
// and it is the defining property of a file the user is about to carry
// somewhere: the reference resolves against *this* machine's keychain, as
// *this* user. Someone moving the file to a runner has to be told that here,
// not by a secret that fails to resolve at 3am.
//
// Some connectors need more than a line. A provider that keeps its key in
// conn.Options never sees a vault reference, so its inventory carries the
// secret itself. The scan's copy of that file is 0600 and deleted when it
// finishes; an exported one is 0600 and stays where it was put, which is a
// materially different exposure -- it can be copied, backed up, or committed.
// The box says so before the write, naming the provider and the flag.
//
// It reads that out of deliverypkg.Locate rather than from a list of connector
// names, and the list is why. Everything here used to name four -- openai,
// ollama, huggingface, claude -- and the installed provider set has thirteen
// such fields across eleven connectors, four of which put a *different*
// credential from the same form somewhere the keychain can hold. A box that
// disclosed from the list would have been silent for datadog's --app-key while
// writing it into the file. Locate looks for the value the launcher sent in the
// asset that came back, so a provider that changes its mind changes this text
// with it.
//
// # Shape
//
// It follows the report viewer's export modal (cli/reportview/export.go) rather
// than inventing a second idiom: a box that takes the body, a destination the
// user can see before pressing the key, O_EXCL so a write never replaces a file
// that is already there, and a one-line footer notice afterwards. The
// difference is what is chosen. The viewer picks a format out of four and
// derives the name; here there is one format and the *path* is what the user
// picks, because the path is where a scheduled job will look for it.

// exportState is the export modal: whether it is open, what the provider made
// of the form, and the name the file will be written under.
//
// It carries its own copy of the connector and the launch request rather than
// reading them off the Model when the write happens. The two are separated by
// however long an OS keychain dialog stays up, and in between the user can
// still be moved off this connector by a provider finishing its install --
// rebuildForm runs underneath whatever is on screen. Writing the inventory that
// was described in the box is the only correct answer to that race.
type exportState struct {
	open bool

	// seq identifies this opening of the box. Both halves of the flow answer
	// asynchronously, and the answer to a box that has since been closed and
	// reopened must not land in the new one.
	seq int

	// asking is true while the connector's provider is being asked what the
	// form means; writing is true while the file is being written.
	asking  bool
	writing bool

	// ready is set once the provider has answered, and is what the enter key
	// waits for. It is not "asking == false": a refusal also ends the asking
	// and leaves nothing to write.
	ready bool

	// name is the path as typed. It is edited in place with the keys rather
	// than by a textinput.Model, the way the value picker's filter is: a second
	// focus-managing widget on a screen that already has one is how the shared
	// input ends up bound to two things at once.
	name string

	connector Connector
	req       launchRequest
	parsed    parsedForm

	// err is the last refusal, shown in the box. It leaves the box open on
	// purpose -- "your export did not happen" is not a message to show for one
	// keystroke, which is what the footer would do with it.
	err error
}

// busy reports whether something is happening that the keys must not interrupt.
func (x exportState) busy() bool { return x.asking || x.writing }

// The prose the box is built from.
const (
	exportTitle = "Export inventory"
	// The subtitle says what the file is *for*, which is the only reason to
	// keep one rather than let the scan delete it.
	exportSubtitle = "a reusable scan: cnspec scan --inventory-file <this file>"

	// exportKeychainNote is the one line the whole feature owes the user. See
	// the package comment above for why it is not optional.
	exportKeychainNote = "The secret stays in this machine's OS keychain and the file only " +
		"references it, so this inventory reproduces the scan on this machine, as this user — " +
		"on another machine or a CI runner the reference will not resolve."

	// exportAmbientNote is what an inventory with no secret in it says instead.
	// It is deliberately not silence: "nothing to warn about" and "we did not
	// look" read the same on screen.
	exportAmbientNote = "This inventory carries no secret: the scan will use whatever credentials " +
		"and files the machine running it already provides."
)

// exportPlaintextNote is the case the one line above is not enough for.
//
// It names the provider and the flag because that is the actionable part -- the
// user can move the key out of the file and into that provider's own
// environment variable afterwards -- and it draws the contrast with the scan's
// own inventory, which is the reason this is worth saying at all: the exposure
// the user has already accepted is not the exposure they are about to create.
func exportPlaintextNote(connector string, flags []string) string {
	return "The " + connector + " provider reads " + flagList(flags) + " as a connection option " +
		"rather than as a credential, so the OS keychain cannot hold it and this file carries the " +
		"secret in plain text. A scan's own inventory is deleted when it finishes; this one stays " +
		"where you put it, so keep it out of version control and off shared machines."
}

// exportNote is one thing the box says about the file before it is written.
type exportNote struct {
	// warn marks a note describing a weaker guarantee than the file's 0600
	// suggests, which is drawn in the warning color.
	warn bool
	text string
}

// notes is what this particular export has to disclose, read from what the
// provider did with the secrets rather than from a table of connector names.
func (x exportState) notes() []exportNote {
	if !x.ready {
		return nil
	}
	var out []exportNote
	if deliverypkg.Keychainable(x.parsed.placed) != nil {
		out = append(out, exportNote{text: exportKeychainNote})
	}
	if plain := x.parsed.plaintextFlags(); len(plain) > 0 {
		out = append(out, exportNote{warn: true, text: exportPlaintextNote(x.connector.Name, plain)})
	}
	if len(out) == 0 {
		out = append(out, exportNote{text: exportAmbientNote})
	}
	return out
}

// defaultExportName is what the box opens with.
//
// Nothing has to be slugged out of it, which is the difference from the report
// viewer's exportSlug: that one derives a file name from an asset name, which
// can be an ARN or a container digest, and this one derives it from a connector
// name -- a flag-shaped identifier the provider declared. What follows it is
// typed by the user, whose target this is to name.
func defaultExportName(c Connector) string { return c.Name + ".inventory.yml" }

// path resolves what was typed into the file that will be written.
func (x exportState) path() (string, error) {
	name := strings.TrimSpace(x.name)
	if name == "" {
		return "", errors.New("give the file a name")
	}

	// ~ is expanded here because there is no shell to expand it: a TUI that
	// takes "~/scans/prod.yml" literally creates a directory called "~" in the
	// working directory and puts the file somewhere the user will never look
	// for it again.
	if name == "~" || name == "~/" {
		return "", errors.New("~ is a directory, not a file name")
	}
	if rest, ok := strings.CutPrefix(name, "~/"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.Wrap(err, "cannot resolve ~")
		}
		name = filepath.Join(home, rest)
	}

	abs, err := filepath.Abs(name)
	if err != nil {
		return "", errors.Wrapf(err, "cannot resolve %s", name)
	}
	// Caught here rather than by the open: O_EXCL on a directory reports EISDIR
	// or EEXIST depending on the platform, and neither says what the user did.
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		return "", errors.Newf("%s is a directory", abs)
	}
	return abs, nil
}

// exportDisplay is a written path as the user would type it again: relative to
// the working directory when it is below it, absolute otherwise.
//
// The footer says the path twice -- what was written, and the command that
// reads it -- and two absolute paths do not fit on a line.
func exportDisplay(path string) string {
	wd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(wd, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return "./" + rel
}

// exportNotice is the footer line after a successful write. It hands over the
// command the file exists for, so the next step is a paste rather than a
// recollection of what --inventory-file is called.
func exportNotice(path string) string {
	shown := exportDisplay(path)
	return "wrote " + shown + " — run it with: cnspec scan --inventory-file " + shown
}

// --- the two asynchronous halves --------------------------------------------

// exportParsedMsg is the provider's reading of the form, on its way back.
type exportParsedMsg struct {
	seq    int
	parsed parsedForm
	err    error
}

// exportWroteMsg is the outcome of a write.
type exportWroteMsg struct {
	seq  int
	path string
	err  error
}

// exportParseCmd asks the connector's provider what the form means, off the
// event loop.
//
// It writes nothing and touches no keychain, which is what makes it safe to run
// on a key press the user has not yet confirmed: opening the box costs a plugin
// subprocess and a gRPC round trip, and nothing else.
func exportParseCmd(seq int, r launchRequest, c Connector) tea.Cmd {
	return func() tea.Msg {
		parsed, err := r.parseForm(c)
		return exportParsedMsg{seq: seq, parsed: parsed, err: err}
	}
}

// exportWriteCmd saves the credential and writes the file, off the event loop
// for the same reason prepareLaunchCmd is: a locked keychain raises an OS
// dialog and does not return until it is answered, and doing that from Update
// freezes the TUI behind a window it is not drawing.
func exportWriteCmd(seq int, r launchRequest, c Connector, parsed parsedForm, path string) tea.Cmd {
	return func() tea.Msg {
		err := r.exportInventory(c, parsed, path)
		return exportWroteMsg{seq: seq, path: path, err: err}
	}
}

// exportInventory saves the credential to the OS keychain and writes the
// inventory that references it.
//
// A keychain failure refuses the export, where the launch path warns and
// carries on. That is not an inconsistency: the launch falls back to a 0600
// file that is deleted when the scan ends, and this one would fall back to a
// file at a path the user chose, holding their password, after a box that had
// just told them the secret stays in the keychain. Silently making that
// sentence false is worse than not writing the file.
func (r launchRequest) exportInventory(c Connector, parsed parsedForm, path string) error {
	secretID, saved, err := r.saveCredential(c, parsed.placed)
	if err != nil {
		return errors.Wrap(err, "the OS keychain would not hold the credential, and the "+
			"exported file must not carry it in plain text instead")
	}
	return deliverypkg.ExportInventory(
		deliverypkg.InventoryFor(c.Name, parsed.asset, saved, secretID), path)
}

// --- frame integration ------------------------------------------------------

// openExport opens the box on the connector under the cursor.
//
// An incomplete form does not export, for the same reason it does not launch:
// what would be written is a target the provider could not make sense of, and
// the useful answer is the field that is missing rather than a file that does
// not scan anything.
func (m Model) openExport() (tea.Model, tea.Cmd) {
	c, ok := m.list.current()
	if !ok {
		return m, nil
	}
	if err := m.detail.form.Validate(); err != nil {
		m.lastErr = err.Error()
		m.focus = focusForm
		m.focusFirstMissing()
		return m, textinput.Blink
	}

	m.export = exportState{
		open:      true,
		asking:    true,
		seq:       m.export.seq + 1,
		name:      defaultExportName(c),
		connector: c,
		req:       m.launchRequestFrom(),
	}
	// The spinner is only ticking while something is loading, so restart it.
	return m, tea.Batch(exportParseCmd(m.export.seq, m.export.req, c), m.spinner.Tick)
}

// exportKey drives the open box. Nothing here reaches a pane and nothing here
// quits: a modal that lets a key end the program underneath it is a modal that
// loses the form the user just filled in.
func (m Model) exportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+e":
		// seq survives the close, so an answer still in flight lands nowhere.
		m.export = exportState{seq: m.export.seq}
		return m, nil
	case "enter":
		return m.startExport()
	case "backspace":
		if !m.export.busy() {
			m.export.name = trimLastRune(m.export.name)
			m.export.err = nil
		}
		return m, nil
	}

	// KeySpace as well as KeyRunes: bubbletea reports a lone space as its own
	// key type with the rune still attached (key.go, "a single KeyRunes or
	// KeySpace event"), so matching only KeyRunes silently drops every space --
	// and a file name with spaces in it is an ordinary thing to want,
	// especially on macOS.
	if !m.export.busy() && (msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace) {
		m.export.name += string(msg.Runes)
		// The name just changed, so whatever the last attempt was refused for
		// is no longer what the box is describing.
		m.export.err = nil
	}
	return m, nil
}

// startExport writes the file, having resolved where it goes first: a path the
// launcher cannot use is a message in the box, not a failed write.
func (m Model) startExport() (tea.Model, tea.Cmd) {
	if m.export.busy() || !m.export.ready {
		return m, nil
	}
	path, err := m.export.path()
	if err != nil {
		m.export.err = err
		return m, nil
	}
	m.export.err = nil
	m.export.writing = true
	return m, tea.Batch(
		exportWriteCmd(m.export.seq, m.export.req, m.export.connector, m.export.parsed, path),
		m.spinner.Tick)
}

// exportParsed takes the provider's answer.
func (m Model) exportParsed(msg exportParsedMsg) (tea.Model, tea.Cmd) {
	if !m.export.open || msg.seq != m.export.seq {
		// The box was closed, or closed and reopened, while the provider was
		// being asked. Nothing was written and nothing was saved, so there is
		// nothing to report.
		return m, nil
	}
	m.export.asking = false
	if msg.err != nil {
		m.export.err = msg.err
		return m, nil
	}
	m.export.parsed = msg.parsed
	m.export.ready = true
	return m, nil
}

// exportWrote takes the outcome of a write.
//
// A completed write is a fact about the filesystem, so it is reported whether
// or not the box that started it is still up -- dismissing the box while an OS
// keychain dialog is on screen is an ordinary thing to do, and a file that
// appeared without a word is worse than one that is announced late.
func (m Model) exportWrote(msg exportWroteMsg) (tea.Model, tea.Cmd) {
	mine := m.export.open && msg.seq == m.export.seq
	if mine && msg.err != nil {
		m.export.writing = false
		m.export.err = msg.err
		return m, nil
	}
	if mine {
		m.export = exportState{seq: m.export.seq}
	}
	if msg.err != nil {
		m.lastErr = "export failed: " + tui.OneLine(msg.err.Error())
		return m, nil
	}
	m.lastNote = exportNotice(msg.path)
	return m, nil
}

// --- rendering --------------------------------------------------------------

// viewExport draws the box, occupying exactly bodyH lines so the footer does
// not move. The geometry is the launcher's other two modals' (modalGeom,
// modalBox, lipgloss.Place), so the three read as one program.
func (m Model) viewExport(l layout) string {
	x := m.export
	boxW, contentW := modalGeom(l.Width)

	var b strings.Builder
	b.WriteString(tui.StyleAccent.Render(tui.Truncate(exportTitle, contentW)))
	b.WriteString("\n")
	b.WriteString(tui.StyleFaint.Render(tui.Truncate(exportSubtitle, contentW)))
	b.WriteString("\n\n")

	if x.asking {
		b.WriteString(m.spinner.View() + " " + tui.StyleDim.Render(tui.Truncate(
			"asking the "+x.connector.Name+" provider what these settings mean…", contentW-2)))
		b.WriteString("\n\n")
	} else if x.ready {
		// The name is a band rather than a bordered box because it is the only
		// thing on this screen the keys can reach, and the launcher already
		// spells "the keys are here" as a band everywhere else.
		b.WriteString(tui.Bar(tui.Truncate("  "+x.name+"▌", contentW), contentW, tui.BandSelected))
		b.WriteString("\n\n")
	}

	for _, note := range x.notes() {
		style := tui.StyleDim
		prefix := "  "
		if note.warn {
			style = tui.StyleWarn
			prefix = "! "
		}
		for i, wrapped := range tui.WrapWords(note.text, contentW-2) {
			if i > 0 {
				prefix = "  "
			}
			b.WriteString(style.Render(prefix + wrapped))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(x.status(m.spinner.View(), contentW))
	b.WriteString("\n\n")
	b.WriteString(tui.StyleFaint.Render(tui.Truncate(x.help(), contentW)))

	box := modalBox.Width(boxW).Render(b.String())
	return lipgloss.Place(l.Width, l.BodyH, lipgloss.Center, lipgloss.Center, box)
}

// status is the line above the key hints: why nothing was written, what is
// being written, or -- the ordinary case -- the exact path the enter key would
// write to, so "where does it go" is answered before the key is pressed.
//
// Every string that came from outside this package goes through tui.OneLine
// first. A provider's refusal is a wrapped error chain from somewhere else and
// several of them contain newlines; truncating to a width does not remove a
// newline, so an unflattened one adds rows to a box whose height the layout has
// already committed to.
func (x exportState) status(spinner string, w int) string {
	switch {
	case x.err != nil:
		return tui.StyleBad.Render(tui.Truncate("! "+tui.OneLine(x.err.Error()), w))
	case x.writing:
		return spinner + " " + tui.StyleDim.Render(tui.Truncate("writing "+x.name+"…", w-2))
	case !x.ready:
		return ""
	}
	path, err := x.path()
	if err != nil {
		return tui.StyleWarn.Render(tui.Truncate("! "+tui.OneLine(err.Error()), w))
	}
	// The head of a path is what goes, not its tail. An absolute path under a
	// deep working directory overflows the box, and cutting the end of it hides
	// the one part the user chose and can check -- the file name.
	return tui.StyleDim.Render("→ " + truncateTail(path, w-2))
}

// help is the key line. It says what enter does rather than naming the action,
// because for the four connectors whose secret cannot go to the keychain what
// enter does is the thing the user is being asked to agree to.
func (x exportState) help() string {
	switch {
	case x.busy():
		return "esc cancel"
	case !x.ready:
		return "esc close"
	case len(x.parsed.plaintextFlags()) > 0:
		// Kept inside the box's content width rather than written out in full:
		// a hint that says what enter agrees to and is then truncated to
		// "…in plain t…" has said nothing. The whole of it is in the note above.
		return "type to name · enter write, secret in plain text · esc cancel"
	default:
		return "type to name the file · enter write · esc cancel"
	}
}

// exportHint is the footer's offer of this box.
//
// It is offered only for a form that could be launched, which is the same rule
// the ^o hint follows: an export of an incomplete target is a file that scans
// nothing, and a key hint for it would be a promise the launcher cannot keep.
func (m Model) exportHint() string {
	if !m.readyToRun() {
		return ""
	}
	return tui.HintSep + tui.Kbd("^e", "export")
}
