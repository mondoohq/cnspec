// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/exceptions"
)

// printExceptionDecisions accounts for every exception read from config files
// for the asset (ADR-0006). The ones in effect name the file they came from and
// how they came to apply; the ones that are not say why, because a committed
// exception next to a failing check, with nothing connecting the two, looks
// like a file that was never read.
func (r *defaultReporter) printExceptionDecisions(assetMrn string) {
	decisions := r.data.GetExceptionDecisions()[assetMrn]
	if decisions == nil || len(decisions.Items) == 0 {
		return
	}

	var inEffect, pending, notInEffect []*policy.ExceptionDecision
	for _, d := range decisions.Items {
		switch d.Outcome {
		case policy.ExceptionOutcome_EXCEPTION_OUTCOME_APPLIED,
			policy.ExceptionOutcome_EXCEPTION_OUTCOME_ACCEPTED,
			policy.ExceptionOutcome_EXCEPTION_OUTCOME_AUTO_ACCEPTED:
			inEffect = append(inEffect, d)
		case policy.ExceptionOutcome_EXCEPTION_OUTCOME_PENDING:
			pending = append(pending, d)
		default:
			notInEffect = append(notInEffect, d)
		}
	}

	r.printDecisionSection("Exceptions from config files [preview]:", inEffect, false)
	r.printDecisionSection("Exceptions awaiting approval (not in effect) [preview]:", pending, true)
	r.printDecisionSection("Exceptions not in effect [preview]:", notInEffect, true)
}

func (r *defaultReporter) printDecisionSection(heading string, decisions []*policy.ExceptionDecision, faint bool) {
	if len(decisions) == 0 {
		return
	}
	sort.SliceStable(decisions, func(i, j int) bool {
		return r.exceptionCheckTitle(decisions[i].Entry) < r.exceptionCheckTitle(decisions[j].Entry)
	})

	r.beginAssetSection()
	r.out(heading + NewLineCharacter)
	for _, d := range decisions {
		e := d.Entry
		line := checkStatus("•", strings.ReplaceAll(exceptions.ActionName(e.Action), "-", " ")) + r.exceptionCheckTitle(e)
		if e.Title != "" {
			line += " (" + e.Title + ")"
		}
		r.out(r.styled(line, faint).Foreground(r.Colors.Disabled).String() + NewLineCharacter)

		details := []string{"from " + exceptionSourceLabel(e)}
		if status := decisionStatus(d); status != "" {
			details = append(details, status)
		}
		if e.ValidUntil != nil {
			details = append(details, "valid until "+time.Unix(e.ValidUntil.Seconds, 0).UTC().Format(time.DateOnly))
		}
		r.out(r.styled("  "+strings.Join(details, " · "), true).String() + NewLineCharacter)
		if e.Justification != "" {
			r.out(r.styled(`  "`+e.Justification+`"`, true).String() + NewLineCharacter)
		}
	}
}

func (r *defaultReporter) exceptionCheckTitle(e *policy.ExceptionEntry) string {
	if r.bundle != nil {
		if q, ok := r.bundle.Queries[e.CheckMrn]; ok && q.Title != "" {
			return q.Title
		}
	}
	if e.CheckUid != "" {
		return e.CheckUid
	}
	return e.CheckMrn
}

// exceptionSourceLabel names the file an exception came from, and the path
// scope that matched, since a file at a repository root can govern many assets
// and its location alone no longer says how far an entry reaches.
func exceptionSourceLabel(e *policy.ExceptionEntry) string {
	label := exceptions.SourceName(e.Source)
	if e.Source.GetRepository() == "" && filepath.IsAbs(label) {
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, label); err == nil && !strings.HasPrefix(rel, "..") {
				label = rel
			}
		}
	}
	if label == "" {
		label = "config"
	}
	if e.Source.GetRef() != "" {
		label += "@" + e.Source.GetRef()
	}
	if e.MatchedPath != "" {
		label += " (" + e.MatchedPath + ")"
	}
	return label
}

func decisionStatus(d *policy.ExceptionDecision) string {
	switch d.Outcome {
	case policy.ExceptionOutcome_EXCEPTION_OUTCOME_APPLIED:
		return "applied locally, no upstream to approve it"
	case policy.ExceptionOutcome_EXCEPTION_OUTCOME_ACCEPTED:
		var names []string
		for _, a := range d.Approvers {
			if a.Email != "" {
				names = append(names, a.Email)
			} else if a.Name != "" {
				names = append(names, a.Name)
			}
		}
		if len(names) == 0 {
			return "approved"
		}
		return "approved by " + strings.Join(names, ", ")
	case policy.ExceptionOutcome_EXCEPTION_OUTCOME_AUTO_ACCEPTED:
		return "auto-accepted, approval is not required"
	case policy.ExceptionOutcome_EXCEPTION_OUTCOME_PENDING:
		return "submitted, pending review"
	case policy.ExceptionOutcome_EXCEPTION_OUTCOME_REJECTED:
		if d.Reason != "" {
			return "rejected: " + d.Reason
		}
		return "rejected"
	case policy.ExceptionOutcome_EXCEPTION_OUTCOME_EXPIRED:
		return "expired"
	case policy.ExceptionOutcome_EXCEPTION_OUTCOME_UNKNOWN_CHECK:
		return "unknown check"
	}
	return d.Reason
}
