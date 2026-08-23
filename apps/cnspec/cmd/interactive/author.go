// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/cnspec/internal/bundle"
	"go.mondoo.com/cnspec/internal/generate"
)

// Authoring a check is the launcher's second verb. The first -- scan -- asks
// what an asset looks like; this one asks what it *should* look like, and
// writes the answer into a bundle the next scan can run.
//
// It is the same flow the line-oriented wizard runs (`policy generate -i`), and
// deliberately so: both drive internal/generate and both write through
// bundle.AppendCheck. What differs is only the surface, which is why the shared
// half lives in those packages rather than beside either front end.

// authorStep is where the authoring pane is in its own small progression.
type authorStep int

const (
	// authorIntent collects what the check should prove.
	authorIntent authorStep = iota
	// authorWorking is the agent running. It is a separate step rather than a
	// flag because the pane draws nothing else while it holds.
	authorWorking
	// authorReview is the generated MQL on screen, awaiting a verdict. This is
	// the human gate the whole design leans on, so it is never skipped -- not
	// even when validation passed.
	authorReview
	// authorDone reports where the check landed.
	authorDone
)

// authorField indexes the intent fields, in the order they are asked.
type authorField int

const (
	fieldTitle authorField = iota
	fieldDesc
	fieldFilter
	fieldFile
	authorFieldCount
)

func (f authorField) label() string {
	switch f {
	case fieldTitle:
		return "Title"
	case fieldDesc:
		return "What must be true"
	case fieldFilter:
		return "Asset filter"
	default:
		return "Write to"
	}
}

func (f authorField) hint() string {
	switch f {
	case fieldTitle:
		return "what the check proves, as a statement"
	case fieldDesc:
		return "the condition, in enough detail to be unambiguous"
	case fieldFilter:
		return "which assets it applies to (proposed from the title)"
	default:
		return "the bundle file the check is appended to"
	}
}

// authorState is the authoring pane: the intent being described, the agent run
// in flight, and the candidate awaiting a verdict.
//
// seq is what makes a late answer safe. Generation outlives the keystroke that
// started it, so a result that arrives after the user moved on has to be
// dropped rather than drawn over whatever is on screen now -- the same reason
// pickerState and exportState carry one.
type authorState struct {
	seq     int
	step    authorStep
	fields  [authorFieldCount]string
	cursor  authorField
	touched bool // the filter was hand-edited, so stop re-proposing one

	gen *generate.Generator

	mql         string
	explanation string
	warn        string // the gate rejected it, but the candidate is still offered
	err         string
	wrote       string
}

// authorGeneratedMsg carries one agent run's outcome back to the event loop.
type authorGeneratedMsg struct {
	seq         int
	mql         string
	explanation string
	warn        string
	err         error
}

// authorWroteMsg reports the write-out.
type authorWroteMsg struct {
	seq  int
	path string
	err  error
}

// authorGenerateCmd runs one generation off the event loop.
//
// GenerateCheck validates internally and retries, so the command returns a
// candidate plus whatever the gate said about it rather than a bare string: a
// query that would not compile is still worth showing, because the reviewer can
// fix it, and hiding it costs another billed run.
func authorGenerateCmd(seq int, gen *generate.Generator, check generate.Check) tea.Cmd {
	return func() tea.Msg {
		res := gen.GenerateCheck(context.Background(), check)
		msg := authorGeneratedMsg{seq: seq}
		switch {
		case res.Action == generate.ActionGenerated:
			msg.mql = bundle.SanitizeText(res.MQL)
			msg.explanation = bundle.SanitizeText(res.Explanation)
		case res.MQL != "":
			// a candidate that failed its gate: offer it, and say so
			msg.mql = bundle.SanitizeText(res.MQL)
			msg.warn = bundle.SanitizeText(res.Reason)
		default:
			msg.err = res.Err
			if msg.err == nil {
				msg.err = errFromReason(bundle.SanitizeText(res.Reason))
			}
		}
		return msg
	}
}

// authorWriteCmd appends the accepted check to its bundle.
func authorWriteCmd(seq int, file, uid, title, desc, filter, mql string) tea.Cmd {
	return func() tea.Msg {
		taken := map[string]bool{}
		if b, err := bundle.LoadFile(file); err == nil {
			taken = bundle.QueryUIDs(b)
		}
		err := bundle.AppendCheck(file, bundle.NextFreeUID(uid, taken), title, desc, filter, mql)
		return authorWroteMsg{seq: seq, path: file, err: err}
	}
}

// openAuthor starts a new authoring session.
func (m Model) openAuthor() (tea.Model, tea.Cmd) {
	gen, err := newAuthorGenerator()
	if err != nil {
		m.lastErr = "cannot author checks: " + err.Error()
		return m, nil
	}

	m.author = authorState{
		seq:  m.author.seq + 1,
		step: authorIntent,
		gen:  gen,
	}
	m.author.fields[fieldFile] = defaultAuthorFile()
	m.phase = phaseAuthoring
	return m, nil
}

// defaultAuthorFile is where a check lands when the user does not say.
func defaultAuthorFile() string {
	return "generated-policy.mql.yaml"
}

// newAuthorGenerator builds the generator the pane drives.
//
// Validation is wired when the machine can do it and left out when it cannot,
// rather than refusing to author: a missing provider means the query cannot be
// compile-checked here, which is a weaker gate, not a reason to stop.
func newAuthorGenerator() (*generate.Generator, error) {
	backend, err := generate.Backend("")
	if err != nil {
		return nil, err
	}
	cfg := generate.Config{Backend: backend, Explain: true}
	if v, verr := generate.NewCompileValidator(); verr == nil {
		cfg.Validator = v
	}
	return generate.New(cfg)
}

// startAuthorGeneration hands the described intent to the agent.
func (m Model) startAuthorGeneration() (tea.Model, tea.Cmd) {
	a := &m.author
	title := strings.TrimSpace(a.fields[fieldTitle])
	if title == "" {
		a.err = "a check needs a title"
		return m, nil
	}

	a.seq++
	a.step = authorWorking
	a.err, a.warn, a.mql, a.explanation = "", "", "", ""

	check := generate.Check{
		UID:   bundle.Slugify(title),
		Title: title,
		Desc:  strings.TrimSpace(a.fields[fieldDesc]),
	}
	if f := strings.TrimSpace(a.fields[fieldFilter]); f != "" {
		check.Filters = []string{f}
	}
	return m, tea.Batch(authorGenerateCmd(a.seq, a.gen, check), m.spinner.Tick)
}

// authorGenerated puts a finished run on screen, if it is still the current one.
func (m Model) authorGenerated(msg authorGeneratedMsg) (tea.Model, tea.Cmd) {
	if msg.seq != m.author.seq || m.author.step != authorWorking {
		return m, nil
	}
	if msg.err != nil {
		m.author.step = authorIntent
		m.author.err = tui.OneLine(msg.err.Error())
		return m, nil
	}
	m.author.step = authorReview
	m.author.mql = msg.mql
	m.author.explanation = msg.explanation
	m.author.warn = msg.warn
	return m, nil
}

// authorWrote reports where the check landed.
func (m Model) authorWrote(msg authorWroteMsg) (tea.Model, tea.Cmd) {
	if msg.seq != m.author.seq {
		return m, nil
	}
	if msg.err != nil {
		m.author.step = authorReview
		m.author.err = tui.OneLine(msg.err.Error())
		return m, nil
	}
	m.author.step = authorDone
	m.author.wrote = msg.path
	return m, nil
}

// acceptAuthored writes the candidate under review.
func (m Model) acceptAuthored() (tea.Model, tea.Cmd) {
	a := &m.author
	if strings.TrimSpace(a.mql) == "" {
		a.err = "there is no query to write"
		return m, nil
	}
	a.seq++
	return m, authorWriteCmd(a.seq,
		strings.TrimSpace(a.fields[fieldFile]),
		bundle.Slugify(a.fields[fieldTitle]),
		strings.TrimSpace(a.fields[fieldTitle]),
		strings.TrimSpace(a.fields[fieldDesc]),
		strings.TrimSpace(a.fields[fieldFilter]),
		a.mql,
	)
}

// closeAuthor returns to the launcher.
func (m Model) closeAuthor() (tea.Model, tea.Cmd) {
	m.author = authorState{seq: m.author.seq}
	m.phase = phaseForm
	return m, nil
}

// proposeFilter fills the filter from the title, unless the user typed one.
//
// It is a proposal rather than a default: the same guess the wizard makes, put
// where it can be seen and corrected before it is used, because a filter naming
// a platform that does not exist matches no asset and fails silently.
func (a *authorState) proposeFilter() {
	if a.touched {
		return
	}
	if f := generate.DefaultFilter(generate.GuessProvider(a.fields[fieldTitle])); f != "" {
		a.fields[fieldFilter] = f
	}
}

// errFromReason turns a bare reason string into an error.
func errFromReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return os.ErrInvalid
	}
	return &authorReasonErr{reason}
}

type authorReasonErr struct{ reason string }

func (e *authorReasonErr) Error() string { return e.reason }

// viewAuthor draws the authoring pane.
func (m Model) viewAuthor(l layout) string {
	a := m.author
	boxW, contentW := modalGeom(l.Width)

	var b strings.Builder
	b.WriteString(tui.StyleAccent.Render(tui.Truncate("Author a check", contentW)))
	b.WriteString("\n" + tui.StyleDim.Render(tui.Truncate(
		"describe what must be true; the agent writes the MQL", contentW)))
	b.WriteString("\n\n")

	switch a.step {
	case authorWorking:
		b.WriteString(m.spinner.View() + " " +
			tui.StyleDim.Render(tui.Truncate("generating…", contentW-2)))
		b.WriteString("\n\n" + tui.StyleFaint.Render(tui.Truncate("esc cancel", contentW)))

	case authorReview, authorDone:
		b.WriteString(tui.StyleDim.Render(tui.Truncate(a.fields[fieldTitle], contentW)) + "\n\n")
		for _, line := range tui.Wrap(a.mql, contentW) {
			b.WriteString(tui.StyleText.Render(line) + "\n")
		}
		if a.explanation != "" {
			b.WriteString("\n")
			for _, line := range tui.WrapWords(a.explanation, contentW) {
				b.WriteString(tui.StyleFaint.Render(line) + "\n")
			}
		}
		if a.warn != "" {
			b.WriteString("\n" + tui.StyleWarn.Render(tui.Truncate("! "+a.warn, contentW)) + "\n")
		}
		b.WriteString("\n")
		if a.step == authorDone {
			b.WriteString(tui.StyleAccent.Render(tui.Truncate("written to "+a.wrote, contentW)))
			b.WriteString("\n\n" + tui.StyleFaint.Render(tui.Truncate(
				"a author another · esc done", contentW)))
		} else {
			b.WriteString(tui.StyleFaint.Render(tui.Truncate(
				"enter accept & write · r regenerate · esc cancel", contentW)))
		}

	default:
		for f := authorField(0); f < authorFieldCount; f++ {
			label := f.label()
			value := a.fields[f]
			line := tui.Truncate(label+": "+value, contentW-2)
			if f == a.cursor {
				b.WriteString(tui.Bar("▸ "+line+"▌", contentW, tui.BandSelected) + "\n")
				continue
			}
			b.WriteString("  " + tui.StyleText.Render(line) + "\n")
		}
		b.WriteString("\n" + tui.StyleFaint.Render(tui.Truncate(a.cursor.hint(), contentW)))
		b.WriteString("\n\n" + tui.StyleFaint.Render(tui.Truncate(
			"↑/↓ move · type to edit · ^g generate · esc cancel", contentW)))
	}

	if a.err != "" {
		b.WriteString("\n" + tui.StyleWarn.Render(tui.Truncate("! "+a.err, contentW)))
	}

	box := modalBox.Width(boxW).Render(b.String())
	return lipgloss.Place(l.Width, l.BodyH, lipgloss.Center, lipgloss.Center, box)
}

// updateAuthoring drives the authoring pane. It owns the screen while it is up,
// so it handles every message rather than falling through to the launcher --
// the same contract phaseScanning and phaseViewing have.
func (m Model) updateAuthoring(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.author.step != authorWorking {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case authorGeneratedMsg:
		return m.authorGenerated(msg)

	case authorWroteMsg:
		return m.authorWrote(msg)

	case tea.KeyMsg:
		return m.authorKey(msg)
	}
	return m, nil
}

// authorKey is the pane's keyboard.
func (m Model) authorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	a := &m.author

	switch a.step {
	case authorWorking:
		// The run is not cancellable mid-flight -- the agent is a child process
		// with its own timeout -- so esc leaves the pane and the late answer is
		// dropped by its sequence number rather than drawn over the launcher.
		if msg.String() == "esc" || msg.String() == "ctrl+c" {
			a.seq++
			return m.closeAuthor()
		}
		return m, nil

	case authorReview:
		switch msg.String() {
		case "enter":
			return m.acceptAuthored()
		case "r":
			return m.startAuthorGeneration()
		case "esc":
			return m.closeAuthor()
		}
		return m, nil

	case authorDone:
		switch msg.String() {
		case "a":
			// another check, same file: keep the destination, drop the intent
			file := a.fields[fieldFile]
			gen := a.gen
			m.author = authorState{seq: a.seq + 1, gen: gen}
			m.author.fields[fieldFile] = file
			return m, nil
		case "esc", "enter":
			return m.closeAuthor()
		}
		return m, nil
	}

	// authorIntent
	switch msg.String() {
	case "esc":
		return m.closeAuthor()
	case "up", "shift+tab":
		if a.cursor > 0 {
			a.cursor--
		}
		return m, nil
	case "down", "tab":
		if a.cursor < authorFieldCount-1 {
			a.cursor++
		}
		return m, nil
	case "ctrl+g", "enter":
		if a.cursor < authorFieldCount-1 && msg.String() == "enter" {
			// enter advances through the fields; only the last one, or ^g
			// anywhere, commits. Generating from a half-filled form wastes a
			// billed run on an intent the user was still writing.
			a.cursor++
			a.proposeFilter()
			return m, nil
		}
		return m.startAuthorGeneration()
	case "backspace":
		a.fields[a.cursor] = trimLastRune(a.fields[a.cursor])
		if a.cursor == fieldFilter {
			a.touched = true
		}
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		a.fields[a.cursor] += string(msg.Runes)
		if a.cursor == fieldFilter {
			a.touched = true
		}
		if a.cursor == fieldTitle {
			a.proposeFilter()
		}
		return m, nil
	}
	if msg.Type == tea.KeySpace {
		a.fields[a.cursor] += " "
		return m, nil
	}
	return m, nil
}
