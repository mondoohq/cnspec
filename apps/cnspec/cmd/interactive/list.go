// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"go.mondoo.com/cnspec/cli/tui"
)

type rowKind int

const (
	rowBlank rowKind = iota
	rowHeader
	rowEntry
)

// row is one line of the left list. Blank separators are rows rather than style
// margins so the line count stays exact; see layout.go.
type row struct {
	kind rowKind
	text string // category name, for rowHeader
	idx  int    // index into listState.filtered, for rowEntry
	sel  int    // index into listState.selectable, for rowEntry; -1 otherwise
}

// listState is the catalog and the way through it: the search over it, the
// rows that search renders to, and where the cursor and the viewport are.
//
// Everything below entries is derived from entries and the search text, and
// refilter is the only thing that derives it. They are one type because they
// were seven fields that had to be recomputed together and nothing but
// convention said so -- setting the search text without calling refilter left
// the rows describing a filter that was no longer in force, and the cursor
// indexing a list that had changed under it.
//
// installing is here rather than beside the pickers because what it names is a
// catalog entry. providers.DefaultProviders strips the flags, argument counts
// and discovery targets a form is built from, so a connector whose provider is
// not on disk is a placeholder; finishing the download replaces it in entries,
// and applyInstalled is the only thing that writes entries after construction.
type listState struct {
	entries []Connector
	search  textinput.Model

	// derived list state
	filtered   []Connector
	rows       []row
	selectable []int
	cursor     int
	// offset scrolls the connector list. See ensureVisible for the one thing
	// the shared type does not do.
	offset tui.Scroll

	// installing names the provider currently being downloaded. Only one runs
	// at a time: the provider cache it writes is not guarded by a lock.
	installing string
}

func newListState(catalog []Connector, height int) listState {
	s := textinput.New()
	s.Placeholder = "filter connectors"
	s.Prompt = "⌕ "
	s.CharLimit = 64
	s.Focus()

	l := listState{entries: catalog, search: s}
	l.refilter(height)
	return l
}

// count is how many connectors the catalog holds, which the header reports.
func (s listState) count() int { return len(s.entries) }

// refilter recomputes the filtered entries and the display rows from the search
// text. Every row it emits is exactly one line.
func (s *listState) refilter(height int) {
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(s.search.Value())))

	s.filtered = s.filtered[:0]
	for _, e := range s.entries {
		if matchesTokens(e.SearchText(), tokens) {
			s.filtered = append(s.filtered, e)
		}
	}

	s.rows = s.rows[:0]
	s.selectable = s.selectable[:0]
	for _, cat := range categoryOrder {
		var idxs []int
		for i, e := range s.filtered {
			if e.Category == cat {
				idxs = append(idxs, i)
			}
		}
		if len(idxs) == 0 {
			continue
		}
		if len(s.rows) > 0 {
			s.rows = append(s.rows, row{kind: rowBlank, sel: -1})
		}
		s.rows = append(s.rows, row{kind: rowHeader, text: cat, sel: -1})
		for _, i := range idxs {
			s.selectable = append(s.selectable, len(s.rows))
			s.rows = append(s.rows, row{kind: rowEntry, idx: i, sel: len(s.selectable) - 1})
		}
	}

	if s.cursor >= len(s.selectable) {
		s.cursor = len(s.selectable) - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
	s.ensureVisible(height)
}

// clearSearch drops the filter and reports whether there was one to drop. The
// answer decides what esc means in the list: emptying the search, or leaving.
func (s *listState) clearSearch(height int) bool {
	if s.search.Value() == "" {
		return false
	}
	s.search.SetValue("")
	s.refilter(height)
	return true
}

// typed hands a key to the search box and refilters behind it.
func (s *listState) typed(msg tea.KeyMsg, height int) tea.Cmd {
	var cmd tea.Cmd
	s.search, cmd = s.search.Update(msg)
	s.refilter(height)
	return cmd
}

// current returns the connector under the cursor.
func (s listState) current() (Connector, bool) {
	if len(s.selectable) == 0 || s.cursor < 0 || s.cursor >= len(s.selectable) {
		return Connector{}, false
	}
	return s.filtered[s.rows[s.selectable[s.cursor]].idx], true
}

// move walks the cursor and scrolls to keep it visible. It reports whether
// there was anything to move to, so a caller does not resynchronise a detail
// pane against a selection that did not change.
func (s *listState) move(delta, height int) bool {
	if len(s.selectable) == 0 {
		return false
	}
	s.cursor = tui.ClampIndex(s.cursor+delta, len(s.selectable))
	s.ensureVisible(height)
	return true
}

// selectRow puts the cursor on a selectable index, reporting whether that index
// exists. A click is hit-tested against a layout computed for an earlier frame,
// so the index it carries is not assumed to still be there.
func (s *listState) selectRow(idx, height int) bool {
	if idx < 0 || idx >= len(s.selectable) {
		return false
	}
	s.cursor = idx
	s.ensureVisible(height)
	return true
}

// ensureVisible scrolls the list so the cursor's row is on screen, pulling in
// the category header above it when there is room.
//
// The clamp is tui.Scroll's, but the walk-back is not and stays here. A row
// scrolled to the very top of the pane is a row that has lost the "☁ CLOUD"
// heading it belongs under, so the offset is nudged up over the header and its
// spacer -- at most two rows, and only while they are not entries themselves.
// tui.Scroll has no idea what a row is and should not learn: this is the
// launcher's list, with the launcher's row kinds, and a shared type carrying a
// special case for one caller is how the duplication this package is being
// pulled out of started.
func (s *listState) ensureVisible(height int) {
	if len(s.selectable) == 0 {
		s.offset.Off = 0
		return
	}
	sel := s.selectable[s.cursor]

	if sel < s.offset.Off {
		s.offset.Off = sel
		// Show the header (and its spacer) above a row scrolled to the top.
		for back := 1; back <= 2 && s.offset.Off > 0 && s.rows[s.offset.Off-1].kind != rowEntry; back++ {
			s.offset.Off--
		}
	}
	if sel >= s.offset.Off+height {
		s.offset.Off = sel - height + 1
	}
	s.offset.Apply(len(s.rows), height)
}

// ensureProvider starts an on-demand install when the selected connector's
// provider is not present. Only one install runs at a time.
func (s *listState) ensureProvider() tea.Cmd {
	c, ok := s.current()
	if !ok || c.Installed || s.installing != "" {
		return nil
	}
	s.installing = c.Provider
	return installProviderCmd(c.Provider)
}

// downloading reports whether a connector's provider is being fetched now.
func (s listState) downloading(c Connector) bool { return s.installing == c.Provider }

// applyInstalled swaps freshly installed connector metadata into the catalog,
// leaving the list and the cursor where the user left them. It reports whether
// the catalog changed, so a caller does not rebuild a detail pane over entries
// that are the ones it already drew.
func (s *listState) applyInstalled(msg providerInstalledMsg, height int) bool {
	if len(msg.conns) == 0 {
		return false
	}
	updated := make(map[string]Connector, len(msg.conns))
	for _, c := range msg.conns {
		updated[c.Name] = c
	}
	for i, e := range s.entries {
		if e.Provider != msg.provider {
			continue
		}
		if c, ok := updated[e.Name]; ok {
			// Keep the category the catalog assigned: it is the launcher's own
			// grouping, not something the provider declares.
			c.Category = e.Category
			s.entries[i] = c
		}
	}
	s.refilter(height)
	return true
}
