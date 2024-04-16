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

func (e *Evidence) convertToPolicyGroup() *PolicyGroup {
	p := &PolicyGroup{
		Uid:   e.Uid,
		Type:  GroupType_CHAPTER,
		Title: e.Title,
	}
	if e.Desc != "" {
		p.Docs = &PolicyGroupDocs{Desc: e.Desc}
	}

	if len(e.Queries) > 0 {
		p.Queries = append(p.Queries, e.Queries...)
	}
	if len(e.Checks) > 0 {
		p.Checks = append(p.Checks, e.Checks...)
	}
	return p
}

// Pulls the framework's control evidences out into a policy. If no evidence is at all present, this function returns nil.
func (f *Framework) generateEvidencePolicy() *Policy {
	policyGroups := []*PolicyGroup{}
	for _, fg := range f.GetGroups() {
		for _, c := range fg.GetControls() {
			if len(c.GetEvidence()) == 0 {
				continue
			}
			for idx, e := range c.GetEvidence() {
				// we use the index as a suffix to ensure no uid collisions
				e.fillUidIfEmpty(f.Uid, c.Uid, fmt.Sprintf("%d", idx))
				policyGroups = append(policyGroups, e.convertToPolicyGroup())
			}
		}
	}

	if len(policyGroups) == 0 {
		return nil
	}

	pol := &Policy{
		Uid:    f.Uid + "-evidence-policy",
		Name:   f.Name + "-evidence-policy",
		Docs:   f.Docs,
		Groups: policyGroups,
	}
	return pol
}

// Pulls the framework's control evidences out into a framework map. If no evidence is at all present, this function returns nil.
func (f *Framework) generateEvidenceFrameworkMap(evidencePolicy *Policy) *FrameworkMap {
	controlMaps := []*ControlMap{}
	for _, fg := range f.GetGroups() {
		for _, c := range fg.GetControls() {
			cm := c.generateEvidenceControlMap()
			if cm != nil {
				controlMaps = append(controlMaps, cm)
			}
		}
	}
	if len(controlMaps) == 0 {
		return nil
	}
	return &FrameworkMap{
		FrameworkOwner:     &explorer.ObjectRef{Uid: f.Uid},
		Uid:                f.Uid + "-evidence-mapping",
		Controls:           controlMaps,
		PolicyDependencies: []*explorer.ObjectRef{{Uid: evidencePolicy.Uid}},
	}
}

// Generates a policy and a framework map from the framework's control evidences.
// If no evidence is present, this function returns nil for both objects.
// The evidence objects are set to nil after this function is called.
func (f *Framework) GenerateEvidenceObjects() (*Policy, *FrameworkMap) {
	evidencePolicy := f.generateEvidencePolicy()
	if evidencePolicy == nil {
		return nil, nil
	}
	evidenceFm := f.generateEvidenceFrameworkMap(evidencePolicy)
	// clear the evidence after we have generated the policy and framework map
	for _, fg := range f.GetGroups() {
		for _, c := range fg.GetControls() {
			c.Evidence = nil
		}
	}
	return evidencePolicy, evidenceFm
}

// Generates a control map by extracting the control's evidence. If no evidence is present, this function returns nil.
func (c *Control) generateEvidenceControlMap() *ControlMap {
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
