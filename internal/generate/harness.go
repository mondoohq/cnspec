// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"context"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v3"
)

// Case is one intent→MQL evaluation case for the generation-quality harness. It
// captures the natural-language intent and how to judge the generated MQL. This
// is the corpus the ADR calls for to detect skill/backend regressions: run it
// offline with a scripted backend in CI, or against a real agent (and a
// --test-target/--test-recording validator) to measure a model.
type Case struct {
	Name     string   `yaml:"name"`
	Intent   string   `yaml:"intent"` // becomes the check title
	Desc     string   `yaml:"desc"`
	Provider string   `yaml:"provider"` // informational
	Filters  []string `yaml:"filters"`
	Props    []Prop   `yaml:"props"`
	// Want, when set, requires the generated MQL to equal this exactly
	// (whitespace-normalized).
	Want string `yaml:"want"`
	// WantContains, when set, requires the generated MQL to contain every listed
	// substring. Use this for robust checks that don't pin exact phrasing.
	WantContains []string `yaml:"wantContains"`
}

// CaseResult is the outcome of evaluating one Case.
type CaseResult struct {
	Case   Case
	Got    string
	Action Action
	Pass   bool
	Reason string
}

// LoadCases reads harness cases from a YAML file: `cases: [ {name, intent, ...} ]`.
func LoadCases(path string) ([]Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not read cases file")
	}
	var doc struct {
		Cases []Case `yaml:"cases"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, errors.Wrap(err, "could not parse cases file")
	}
	return doc.Cases, nil
}

// RunCases generates MQL for every case with the given generator and evaluates
// the result. It is deterministic given a deterministic backend, so it doubles
// as an offline regression test and an online model evaluation.
func RunCases(ctx context.Context, g *Generator, cases []Case) []CaseResult {
	out := make([]CaseResult, 0, len(cases))
	for _, c := range cases {
		res := g.GenerateCheck(ctx, c.toCheck())
		out = append(out, evaluateCase(c, res))
	}
	return out
}

// PassRate returns the fraction of results that passed, and the pass/total
// counts.
func PassRate(results []CaseResult) (rate float64, passed, total int) {
	total = len(results)
	for _, r := range results {
		if r.Pass {
			passed++
		}
	}
	if total > 0 {
		rate = float64(passed) / float64(total)
	}
	return rate, passed, total
}

func (c Case) toCheck() Check {
	return Check{
		UID:     c.Name,
		Title:   c.Intent,
		Desc:    c.Desc,
		Filters: c.Filters,
		Props:   c.Props,
	}
}

func evaluateCase(c Case, r Result) CaseResult {
	cr := CaseResult{Case: c, Got: r.MQL, Action: r.Action}

	if r.Action != ActionGenerated {
		cr.Reason = "not generated: " + r.Reason
		return cr
	}

	if c.Want != "" {
		if normalizeMQL(r.MQL) == normalizeMQL(c.Want) {
			cr.Pass = true
		} else {
			cr.Reason = "does not match expected MQL"
		}
		return cr
	}

	if len(c.WantContains) > 0 {
		var missing []string
		for _, sub := range c.WantContains {
			if !strings.Contains(r.MQL, sub) {
				missing = append(missing, sub)
			}
		}
		if len(missing) == 0 {
			cr.Pass = true
		} else {
			cr.Reason = "missing expected substrings: " + strings.Join(missing, ", ")
		}
		return cr
	}

	// no expectation beyond "it generated and validated"
	cr.Pass = true
	return cr
}

// normalizeMQL collapses whitespace so exact matches ignore trivial formatting.
func normalizeMQL(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
