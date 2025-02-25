// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"fmt"

	"go.mondoo.com/cnquery/v11/explorer"
)

func (e *Evidence) fillUidIfEmpty(frameworkUid string, controlUid string, suffix string) {
	// if we have an uid already set, simply return
	if e.Uid != "" {
		return
	}
	e.Uid = fmt.Sprintf("%s-%s-evidence-%s", frameworkUid, controlUid, suffix)
}

func (e *Evidence) convertToPolicyGroup() *PolicyGroup {
	queries := []*explorer.Mquery{}
	checks := []*explorer.Mquery{}

	for _, q := range e.Queries {
		if q.Mrn == "" {
			queries = append(queries, q)
		}
	}
	for _, c := range e.Checks {
		if c.Mrn == "" {
			checks = append(checks, c)
		}
	}

	// no queries or checks, we don't need a policy group
	if len(queries) == 0 && len(checks) == 0 {
		return nil
	}

	p := &PolicyGroup{
		Uid:     e.Uid,
		Type:    GroupType_CHAPTER,
		Title:   e.Title,
		Queries: queries,
		Checks:  checks,
	}
	if e.Desc != "" {
		p.Docs = &PolicyGroupDocs{Desc: e.Desc}
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
				polGrp := e.convertToPolicyGroup()
				if polGrp != nil {
					policyGroups = append(policyGroups, polGrp)
				}
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
	fm := &FrameworkMap{
		FrameworkOwner: &explorer.ObjectRef{Uid: f.Uid},
		Uid:            f.Uid + "-evidence-mapping",
		Controls:       controlMaps,
	}
	if evidencePolicy != nil {
		fm.PolicyDependencies = []*explorer.ObjectRef{{Uid: evidencePolicy.Uid}}
	}
	return fm
}

// Generates a policy and a framework map from the framework's control evidences.
// If no evidence is present, this function returns nil for both objects.
// The evidence objects are set to nil after this function is called.
func (f *Framework) GenerateEvidenceObjects() (*Policy, *FrameworkMap) {
	evidencePolicy := f.generateEvidencePolicy()
	evidenceFm := f.generateEvidenceFrameworkMap(evidencePolicy)
	// clear the evidence after we have generated the policy and framework map
	for _, fg := range f.GetGroups() {
		for _, c := range fg.GetControls() {
			c.Evidence = nil
		}
	}
	if evidenceFm != nil {
		for _, dep := range f.Dependencies {
			evidenceFm.FrameworkDependencies = append(evidenceFm.FrameworkDependencies, &explorer.ObjectRef{Mrn: dep.Mrn})
		}
	}
	return evidencePolicy, evidenceFm
}

// Generates a control map by extracting the control's evidence. If no evidence is present, this function returns nil.
func (ctrl *Control) generateEvidenceControlMap() *ControlMap {
	// if no evidence, we don't need a control map
	if len(ctrl.GetEvidence()) == 0 {
		return nil
	}

	checkRefs := []*ControlRef{}
	queryRefs := []*ControlRef{}
	controlRefs := []*ControlRef{}

	for _, e := range ctrl.GetEvidence() {
		for _, ch := range e.Checks {
			if ch.Mrn != "" {
				checkRefs = append(checkRefs, &ControlRef{Mrn: ch.Mrn})
			} else {
				checkRefs = append(checkRefs, &ControlRef{Uid: ch.Uid})
			}
		}
		for _, q := range e.Queries {
			if q.Mrn != "" {
				queryRefs = append(queryRefs, &ControlRef{Mrn: q.Mrn})
			} else {
				queryRefs = append(queryRefs, &ControlRef{Uid: q.Uid})
			}
		}
		for _, q := range e.Controls {
			if q.Mrn != "" {
				controlRefs = append(controlRefs, &ControlRef{Mrn: q.Mrn})
			} else {
				controlRefs = append(controlRefs, &ControlRef{Uid: q.Uid})
			}
		}
	}

	return &ControlMap{
		Uid:      ctrl.Uid,
		Checks:   checkRefs,
		Queries:  queryRefs,
		Controls: controlRefs,
	}
}
