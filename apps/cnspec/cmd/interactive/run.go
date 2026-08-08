// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Result is the outcome of an interactive launcher session.
type Result struct {
	// Launched is true when the user picked a command to run.
	Launched bool
	// Args is the assembled command line, e.g. ["scan", "aws", "--discover", "all"].
	Args []string
}

// Run shows the interactive launcher and blocks until the user picks a command
// or quits. When Result.Launched is true, Result.Args holds the command to run.
func Run() (Result, error) {
	catalog := BuildCatalog()
	m := NewModel(catalog)

	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInputTTY())
	final, err := prog.Run()
	if err != nil {
		return Result{}, err
	}

	fm, ok := final.(Model)
	if !ok || !fm.launched {
		return Result{Launched: false}, nil
	}
	return Result{Launched: true, Args: fm.result}, nil
}

// tokenize splits a user-entered argument string into individual args,
// honoring double and single quotes so values like -c "aws.regions" survive as
// a single token. This is intentionally small: it is a convenience for the
// launcher, not a full shell parser.
func tokenize(s string) []string {
	var out []string
	var cur []rune
	var quote rune
	inToken := false

	flush := func() {
		if inToken {
			out = append(out, string(cur))
			cur = cur[:0]
			inToken = false
		}
	}

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur = append(cur, r)
			}
			inToken = true
		case r == '"' || r == '\'':
			quote = r
			inToken = true
		case r == ' ' || r == '\t':
			flush()
		default:
			cur = append(cur, r)
			inToken = true
		}
	}
	flush()
	return out
}
