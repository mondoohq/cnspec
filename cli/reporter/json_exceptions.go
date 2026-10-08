// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"strings"
	"time"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/exceptions"
)

// convertExceptionDecisions lists, per asset, every exception read from config
// files and what became of it (ADR-0006). It returns nil when there are none,
// so a report without exceptions keeps its shape.
func convertExceptionDecisions(in map[string]*policy.ExceptionDecisions) map[string]*ExceptionDecisions {
	if len(in) == 0 {
		return nil
	}
	res := make(map[string]*ExceptionDecisions, len(in))
	for assetMrn, decisions := range in {
		items := make([]*ExceptionDecision, 0, len(decisions.GetItems()))
		for _, d := range decisions.GetItems() {
			e := d.GetEntry()
			check := e.GetCheckUid()
			if check == "" {
				check = e.GetCheckMrn()
			}
			item := &ExceptionDecision{
				Check:         check,
				CheckMrn:      e.GetCheckMrn(),
				Action:        exceptions.ActionName(e.GetAction()),
				Outcome:       outcomeName(d.GetOutcome()),
				Reason:        d.GetReason(),
				Title:         e.GetTitle(),
				Justification: e.GetJustification(),
				Paths:         e.GetPaths(),
				MatchedPath:   e.GetMatchedPath(),
				Source: &ExceptionSource{
					Scope:      e.GetSource().GetScope(),
					Provider:   e.GetSource().GetProvider(),
					Repository: e.GetSource().GetRepository(),
					Ref:        e.GetSource().GetRef(),
					Path:       e.GetSource().GetPath(),
				},
			}
			if e.GetValidUntil() != nil {
				item.ValidUntil = time.Unix(e.GetValidUntil().GetSeconds(), 0).UTC().Format(time.RFC3339)
			}
			for _, a := range d.GetApprovers() {
				if a.GetEmail() != "" {
					item.Approvers = append(item.Approvers, a.GetEmail())
				} else if a.GetName() != "" {
					item.Approvers = append(item.Approvers, a.GetName())
				}
			}
			items = append(items, item)
		}
		res[assetMrn] = &ExceptionDecisions{Items: items}
	}
	return res
}

// outcomeName is the outcome without its enum prefix, e.g. "auto-accepted".
func outcomeName(o policy.ExceptionOutcome) string {
	name := strings.TrimPrefix(o.String(), "EXCEPTION_OUTCOME_")
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}
