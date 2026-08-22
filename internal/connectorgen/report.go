// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connectorgen

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
)

// WriteJSON renders the artifact.
//
// Indented and newline-terminated, because it is checked in and its diff is
// what a reviewer reads. The ordering is settled in Extract; nothing here
// re-sorts, so a difference in this file is a difference in the source it was
// generated from.
func WriteJSON(w io.Writer, art *Artifact) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(art); err != nil {
		return errors.Wrap(err, "cannot encode the connector artifact")
	}
	return nil
}

// Summary counts what came out, per gap kind.
type Summary struct {
	Connectors int
	// WithEnv is how many connectors got at least one flag-to-variable
	// association.
	WithEnv int
	// EnvVars is how many distinct environment variables were attributed.
	EnvVars int
	// WithPositional is how many connectors got a sub-command vocabulary.
	WithPositional int
	// CarriedForward is how many connectors came from the previous artifact
	// because the source read here does not cover them.
	CarriedForward int
	// GapsByKind counts the gaps, which is the number that decides whether the
	// metadata needs an SDK change before anything is built on it.
	GapsByKind map[string]int
}

// Summarise counts the artifact.
func Summarise(art *Artifact) Summary {
	s := Summary{Connectors: len(art.Connectors), GapsByKind: map[string]int{}}
	vars := map[string]bool{}
	for _, c := range art.Connectors {
		if len(c.Env) > 0 {
			s.WithEnv++
		}
		for _, fe := range c.Env {
			for _, v := range fe.Vars {
				vars[v] = true
			}
		}
		if len(c.Positional) > 0 {
			s.WithPositional++
		}
		if c.CarriedForward {
			s.CarriedForward++
		}
	}
	s.EnvVars = len(vars)
	for _, g := range art.Gaps {
		s.GapsByKind[g.Kind]++
	}
	return s
}

// WriteReport prints the extraction summary and the gap list.
//
// The gap list is the output that matters. Everything the artifact does carry
// is also recoverable from an installed provider or was already hand-written;
// what could not be determined is the part that is new, and it is the
// specification for what the provider SDK would have to declare for cnspec to
// stop reading Go source at all.
func WriteReport(w io.Writer, art *Artifact) {
	s := Summarise(art)

	fmt.Fprintf(w, "connectors:      %d\n", s.Connectors)
	fmt.Fprintf(w, "with env vars:   %d connectors, %d distinct variables\n", s.WithEnv, s.EnvVars)
	fmt.Fprintf(w, "with positional: %d connectors\n", s.WithPositional)
	if s.CarriedForward > 0 {
		fmt.Fprintf(w, "carried forward:  %d connectors the source no longer covers\n", s.CarriedForward)
	}
	for _, src := range art.Sources {
		commit := src.Commit
		if commit == "" {
			commit = "(not a git checkout)"
		} else if len(commit) > 12 {
			commit = commit[:12]
		}
		dirty := ""
		if src.Dirty {
			dirty = " (dirty)"
		}
		fmt.Fprintf(w, "source %-12s %s%s, %d providers\n", src.Name, commit, dirty, src.Providers)
	}

	fmt.Fprintf(w, "\ngaps: %d\n", len(art.Gaps))
	kinds := make([]string, 0, len(s.GapsByKind))
	for k := range s.GapsByKind {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		fmt.Fprintf(w, "  %-26s %d\n", k, s.GapsByKind[k])
	}

	for _, k := range kinds {
		fmt.Fprintf(w, "\n== %s ==\n", k)
		for _, g := range art.Gaps {
			if g.Kind != k {
				continue
			}
			subject := g.Provider
			if g.Connector != "" && g.Connector != g.Provider {
				subject += "/" + g.Connector
			}
			if subject == "" {
				subject = "-"
			}
			line := fmt.Sprintf("  %-24s %s", subject, g.Detail)
			if g.Where != "" {
				line += "  [" + g.Where + "]"
			}
			fmt.Fprintln(w, strings.TrimRight(line, " "))
		}
	}
}
