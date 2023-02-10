package policy

import (
	"regexp"
	"strings"

	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/mrn"
)

// FIXME: DEPRECATED, remove in v9.0
// This file contains conversion and helper structures that were introduced
// with the PolicyV2 update in late v7.x. The can be safely removed (alongside
// the old proto structures) in v9.

func (d *DeprecatedV7_Bundle) ToV8() *Bundle {
	if d == nil {
		return nil
	}

	FixZeroValuesInPolicyBundle(d)

	res := Bundle{
		OwnerMrn: d.OwnerMrn,
		Policies: make([]*Policy, len(d.Policies)),
		Queries:  make([]*explorer.Mquery, len(d.Queries)),
		Props:    deprecatedV7_Mqueries(d.Props).ToV8Props(),
		Docs:     d.Docs,
	}

	for i := range d.Policies {
		res.Policies[i] = d.Policies[i].ToV8()
	}

	props := make(map[string]*explorer.Property, len(d.Props))
	for i := range res.Props {
		prop := res.Props[i]
		props[prop.Uid] = prop
	}

	for i := range d.Queries {
		cur := d.Queries[i].ToV8()
		updateProps(cur, props)
		res.Queries[i] = cur
	}

	return &res
}

var reMqlProperty = regexp.MustCompile("props\\.[a-zA-Z0-9]+")

func updateProps(q *explorer.Mquery, lookup map[string]*explorer.Property) {
	names := reMqlProperty.FindAllString(q.Mql, -1)
	for i := range names {
		name := names[i][6:]
		if _, ok := lookup[name]; ok {
			q.Props = append(q.Props, &explorer.Property{
				Uid: name,
			})
		}
	}
}

func (s *DeprecatedV7_SeverityValue) ToV8() *explorer.Impact {
	if s == nil {
		return nil
	}
	return &explorer.Impact{
		Value:   int32(s.Value),
		Weight:  -1,
		Scoring: explorer.Impact_SCORING_UNSPECIFIED,
	}
}

type deprecatedV7_Mqueries []*DeprecatedV7_Mquery

func (d deprecatedV7_Mqueries) ToV8Props() []*explorer.Property {
	if len(d) == 0 {
		return nil
	}

	res := make([]*explorer.Property, len(d))
	for i := range d {
		cur := d[i]
		res[i] = cur.ToV8Prop()
	}
	return res
}

type deprecatedV7_MqueryRefs []*DeprecatedV7_MqueryRef

func (d deprecatedV7_MqueryRefs) ToV8() []*explorer.MqueryRef {
	if len(d) == 0 {
		return nil
	}

	res := make([]*explorer.MqueryRef, len(d))
	for i := range d {
		res[i] = d[i].ToV8()
	}
	return res
}

func (d *DeprecatedV7_MqueryRef) ToV8() *explorer.MqueryRef {
	if d == nil {
		return nil
	}

	return &explorer.MqueryRef{
		Title: d.Title,
		Url:   d.Url,
	}
}

func (d *DeprecatedV7_MqueryDocs) ToV8() *explorer.MqueryDocs {
	if d == nil {
		return nil
	}

	return &explorer.MqueryDocs{
		Desc:  d.Desc,
		Audit: d.Audit,
		Remediation: &explorer.Remediation{
			Items: []*explorer.TypedDoc{{
				Id:   "default",
				Desc: d.Remediation,
			}},
		},
	}
}

func (d *DeprecatedV7_Mquery) ToV8() *explorer.Mquery {
	if d == nil {
		return nil
	}

	return &explorer.Mquery{
		Mql:      d.Query,
		CodeId:   d.CodeId,
		Checksum: d.Checksum,
		Mrn:      d.Mrn,
		Uid:      d.Uid,
		Type:     d.Type,
		Impact:   d.Severity.ToV8(),
		Title:    d.Title,
		Docs:     d.Docs.ToV8(),
		Refs:     deprecatedV7_MqueryRefs(d.Refs).ToV8(),
		Tags:     d.Tags,
	}
}

func (d *DeprecatedV7_Mquery) ToV8Prop() *explorer.Property {
	if d == nil {
		return nil
	}

	return &explorer.Property{
		Mql:      d.Query,
		CodeId:   d.CodeId,
		Checksum: d.Checksum,
		Mrn:      d.Mrn,
		Uid:      d.Uid,
		Type:     d.Type,
		Title:    d.Title,
	}
}

type deprecatedV7_Authors []*DeprecatedV7_Author

func (d deprecatedV7_Authors) ToV8() []*explorer.Author {
	if len(d) == 0 {
		return nil
	}

	res := make([]*explorer.Author, len(d))
	for i := range d {
		res[i] = d[i].ToV8()
	}
	return res
}

func (d *DeprecatedV7_Author) ToV8() *explorer.Author {
	if d == nil {
		return nil
	}

	return &explorer.Author{
		Name:  d.Name,
		Email: d.Email,
	}
}

type DeprecatedV7_Props map[string]string

func (d DeprecatedV7_Props) ToV8() []*explorer.Property {
	if len(d) == 0 {
		return nil
	}

	res := make([]*explorer.Property, len(d))
	i := 0
	for key := range d {
		res[i] = &explorer.Property{
			Uid: key,
		}
		i++
	}
	return res
}

type deprecatedV7_AssetFilters map[string]*DeprecatedV7_Mquery

func (d deprecatedV7_AssetFilters) ToV8() *explorer.Filters {
	if len(d) == 0 {
		return nil
	}

	res := explorer.Filters{
		Items: make(map[string]*explorer.Mquery, len(d)),
	}
	for k, v := range d {
		res.Items[k] = v.ToV8()
	}
	return &res
}

type deprecatedV7_PolicySpecs []*DeprecatedV7_PolicySpec

func (d deprecatedV7_PolicySpecs) ToV8() []*PolicyGroup {
	if d == nil {
		return nil
	}

	res := make([]*PolicyGroup, len(d))
	for i := range d {
		res[i] = d[i].ToV8()
	}
	return res
}

func Impact2ScoringSpec(impact *explorer.Impact, action QueryAction) *DeprecatedV7_ScoringSpec {
	if impact == nil {
		return nil
	}

	weight := impact.Weight
	if weight == -1 {
		weight = 1
	}

	var severity *DeprecatedV7_SeverityValue
	if impact.Value != -1 {
		severity = &DeprecatedV7_SeverityValue{Value: int64(impact.Value)}
	}

	return &DeprecatedV7_ScoringSpec{
		Weight:             uint32(weight),
		WeightIsPercentage: false,
		ScoringSystem:      ScoringSystem(impact.Scoring), // numbers are identical in this enum
		Action:             action,
		Severity:           severity,
	}
}

func (s *DeprecatedV7_ScoringSpec) ApplyToV8(ref *explorer.Mquery) {
	// For convenience we allow calling it on nil and handle it here.
	if s == nil {
		return
	}

	// If the action is unspecified, it means that the spec is effectively null.
	// Since it's null, don't do anything with it.
	if s.Action == QueryAction_UNSPECIFIED {
		return
	}

	ref.Action = explorer.Mquery_Action(s.Action)

	// For deactivate we don't need anything else in the spec. Just turn it off and
	// we are done.
	if s.Action == QueryAction_DEACTIVATE {
		return
	}

	if ref.Impact == nil {
		ref.Impact = &explorer.Impact{}
	}
	ref.Impact.Scoring = explorer.Impact_ScoringSystem(s.ScoringSystem)
	ref.Impact.Weight = int32(s.Weight)
}

func (d *DeprecatedV7_PolicySpec) ToV8() *PolicyGroup {
	policies := make([]*PolicyRef, len(d.Policies))
	i := 0
	for id, spec := range d.Policies {
		ref := &PolicyRef{}

		if spec != nil {
			ref.Action = PolicyRef_Action(spec.Action)
		}

		if strings.HasPrefix(id, "//") && mrn.IsValid(id) {
			ref.Mrn = id
		} else {
			ref.Uid = id
		}

		policies[i] = ref
		i++
	}

	checks := make([]*explorer.Mquery, len(d.ScoringQueries))
	i = 0
	for id, spec := range d.ScoringQueries {
		ref := &explorer.Mquery{}
		spec.ApplyToV8(ref)

		if strings.HasPrefix(id, "//") && mrn.IsValid(id) {
			ref.Mrn = id
		} else {
			ref.Uid = id
		}

		checks[i] = ref
		i++
	}

	queries := make([]*explorer.Mquery, len(d.DataQueries))
	i = 0
	for id, action := range d.DataQueries {
		ref := &explorer.Mquery{}

		if action != QueryAction_UNSPECIFIED {
			ref.Action = explorer.Mquery_Action(action)
		}

		if strings.HasPrefix(id, "//") && mrn.IsValid(id) {
			ref.Mrn = id
		} else {
			ref.Uid = id
		}

		queries[i] = ref
		i++
	}

	var filters *explorer.Filters
	if d.AssetFilter != nil {
		filters = &explorer.Filters{
			Items: map[string]*explorer.Mquery{},
		}

		filter := d.AssetFilter.ToV8()
		// the key is a placeholder that will be replaced once this is compiled
		// for the first time
		filters.Items["default"] = filter
	}

	return &PolicyGroup{
		Policies: policies,
		Checks:   checks,
		Queries:  queries,
		Filters:  filters,

		StartDate:    d.StartDate,
		EndDate:      d.EndDate,
		ReminderDate: d.ReminderDate,
		Title:        d.Title,
		Docs:         d.Docs,
		Created:      d.Created,
		Modified:     d.Modified,
	}
}

func (d *DeprecatedV7_Policy) ToV8() *Policy {
	if d == nil {
		return nil
	}

	return &Policy{
		Mrn:           d.Mrn,
		Uid:           d.Uid,
		Name:          d.Name,
		Version:       d.Version,
		OwnerMrn:      d.OwnerMrn,
		License:       "unspecified",
		Docs:          d.Docs,
		ScoringSystem: d.ScoringSystem,
		Authors:       deprecatedV7_Authors(d.Authors).ToV8(),
		Created:       d.Created,
		Modified:      d.Modified,
		Tags:          d.Tags,
		Props:         DeprecatedV7_Props(d.Props).ToV8(),
		Filters:       deprecatedV7_AssetFilters(d.AssetFilters).ToV8(),
		QueryCounts:   d.QueryCounts,

		Groups: deprecatedV7_PolicySpecs(d.Specs).ToV8(),

		LocalContentChecksum:   d.LocalContentChecksum,
		GraphContentChecksum:   d.GraphContentChecksum,
		LocalExecutionChecksum: d.LocalExecutionChecksum,
		GraphExecutionChecksum: d.GraphExecutionChecksum,
	}
}

// fixme - this is a hack to deal with the fact that zero valued ScoringSpecs are getting deserialized
// instead of nil pointers for ScoringSpecs.
// This is a quick fix for https://gitlab.com/mondoolabs/mondoo/-/issues/455
// so that we can get a fix out while figuring out wtf is up with our null pointer serialization
// open issue for deserialization: https://gitlab.com/mondoolabs/mondoo/-/issues/508
func FixZeroValuesInPolicyBundle(bundle *DeprecatedV7_Bundle) {
	for _, policy := range bundle.Policies {
		for _, spec := range policy.Specs {
			if spec.Policies != nil {
				for k, v := range spec.Policies {
					// v.Action is only 0 for zero value structs
					if v != nil && v.Action == 0 {
						spec.Policies[k] = nil
					}
				}
			}

			if spec.ScoringQueries != nil {
				for k, v := range spec.ScoringQueries {
					// v.Action is only 0 for zero value structs
					if v != nil && v.Action == 0 {
						spec.ScoringQueries[k] = nil
					}
				}
			}
		}
	}
}
