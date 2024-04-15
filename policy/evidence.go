// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"fmt"

	"go.mondoo.com/cnquery/v10/explorer"
)

func (e *Evidence) fillUidIfEmpty(frameworkUid string, controlUid string, suffix string) {
	// if we have an uid already set, simply return
	if e.Uid != "" {
		return
	}
	e.Uid = fmt.Sprintf("%s-%s-evidence-%s", frameworkUid, controlUid, suffix)
}

func (e *Evidence) convertToPolicy() *Policy {
	p := &Policy{
		Uid:  e.Uid + "-policy",
		Name: e.Title + "-policy",
	}
	if e.Desc != "" {
		p.Docs = &PolicyDocs{Desc: e.Desc}
	}
	if len(e.Queries) > 0 {
		queriesG := &PolicyGroup{
			Queries: e.Queries,
			Type:    GroupType_CHAPTER,
			Uid:     "evidence-queries",
		}
		p.Groups = append(p.Groups, queriesG)
	}
	if len(e.Checks) > 0 {
		checksG := &PolicyGroup{
			Checks: e.Checks,
			Type:   GroupType_CHAPTER,
			Uid:    "evidence-checks",
		}
		p.Groups = append(p.Groups, checksG)
	}
	return p
}

// Pulls the control's evidences out into policies. If no evidence is present, this function returns nil.
func (c *Control) GenerateEvidencePolicies(frameworkUid string) []*Policy {
	// if no evidence, we dont need policies
	if len(c.GetEvidence()) == 0 {
		return nil
	}
	policies := []*Policy{}
	for idx, e := range c.GetEvidence() {
		// we use the index as a suffix to ensure no uid collisions
		e.fillUidIfEmpty(frameworkUid, c.Uid, fmt.Sprintf("%d", idx))
		policies = append(policies, e.convertToPolicy())
	}
	return policies
}

func GenerateFrameworkMap(controlMaps []*ControlMap, framework *Framework, policies []*Policy) *FrameworkMap {
	policyRefs := []*explorer.ObjectRef{}
	for _, p := range policies {
		policyRefs = append(policyRefs, &explorer.ObjectRef{Uid: p.Uid})
	}
	return &FrameworkMap{
		FrameworkOwner:     &explorer.ObjectRef{Uid: framework.Uid},
		Uid:                framework.Uid + "-evidence-mapping",
		Controls:           controlMaps,
		PolicyDependencies: policyRefs,
	}
}

// Generates a control map by extracting the control's evidence. If no evidence is present, this function returns nil.
func (c *Control) GenerateEvidenceControlMap() *ControlMap {
	// if no evidence, we dont need a control map
	if len(c.GetEvidence()) == 0 {
		return nil
	}

	checkRefs := []*ControlRef{}
	queryRefs := []*ControlRef{}

	for _, e := range c.GetEvidence() {
		for c := range e.Checks {
			checkRefs = append(checkRefs, &ControlRef{Uid: e.Checks[c].Uid})
		}

		for q := range e.Queries {
			queryRefs = append(queryRefs, &ControlRef{Uid: e.Queries[q].Uid})
		}
	}

	return &ControlMap{
		Uid:     c.Uid,
		Checks:  checkRefs,
		Queries: queryRefs,
	}
}
