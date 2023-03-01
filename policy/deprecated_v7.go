package policy

import (
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/sortx"
)

// FIXME: DEPRECATED, remove in v9.0 (all of it)
// This file contains conversion and helper structures that were introduced
// with the PolicyV2 update in late v7.x. The can be safely removed (alongside
// the old proto structures) in v9.

// DeprecatedV7Conversions will find any v7 pieces in the bundle and convert
// them to v8+
func (p *Bundle) DeprecatedV7Conversions() {
	p.deprecatedV7convertQueries()
	p.deprecatedV7convertPolicies()
}

// Find any v7 policies and convert them to v8+
// Note: we don't want to duplicate policies; If it exists in v7 and in v8,
// then v7 policies are dropped. Checks for both UIDs and MRNs
func (p *Bundle) deprecatedV7convertPolicies() {
	if len(p.DeprecatedV7Policies) == 0 {
		return
	}

	existing := map[string]struct{}{}
	for i := range p.Policies {
		cur := p.Policies[i]
		if cur.Uid != "" {
			existing[cur.Uid] = struct{}{}
		}
		if cur.Mrn != "" {
			existing[cur.Mrn] = struct{}{}
		}
	}

	for i := range p.DeprecatedV7Policies {
		cur := p.DeprecatedV7Policies[i]
		if _, ok := existing[cur.Mrn]; ok {
			continue
		}
		if _, ok := existing[cur.Uid]; ok {
			continue
		}

		p.Policies = append(p.Policies, cur.ToV8())
	}

	p.DeprecatedV7Policies = nil
}

// Find any v7 queries and convert them to v8+
// Note: we don't want to duplicate queries; If it exists in v7 and in v8,
// then v7 queries are dropped. Checks for both UIDs and MRNs and across all
// policies (which is why we run this before the policy conversion)
func (p *Bundle) deprecatedV7convertQueries() {
	if len(p.DeprecatedV7Queries) == 0 {
		return
	}

	existing := map[string]struct{}{}
	for i := range p.Queries {
		cur := p.Queries[i]
		if cur.Uid != "" {
			existing[cur.Uid] = struct{}{}
		}
		if cur.Mrn != "" {
			existing[cur.Mrn] = struct{}{}
		}
	}

	for i := range p.Policies {
		pol := p.Policies[i]
		for j := range pol.Groups {
			group := pol.Groups[j]
			for k := range group.Queries {
				cur := group.Queries[k]
				if cur.Uid != "" {
					existing[cur.Uid] = struct{}{}
				}
				if cur.Mrn != "" {
					existing[cur.Mrn] = struct{}{}
				}
			}
			for k := range group.Checks {
				cur := group.Checks[k]
				if cur.Uid != "" {
					existing[cur.Uid] = struct{}{}
				}
				if cur.Mrn != "" {
					existing[cur.Mrn] = struct{}{}
				}
			}
		}
	}

	for i := range p.DeprecatedV7Queries {
		cur := p.DeprecatedV7Queries[i]
		if _, ok := existing[cur.Mrn]; ok {
			continue
		}
		if _, ok := existing[cur.Uid]; ok {
			continue
		}

		p.Queries = append(p.Queries, cur.ToV8())
	}

	p.DeprecatedV7Queries = nil
}

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
	if ref == nil {
		log.Error().Msg("cannot apply v7 scoring spec to mquery, query is nil")
		return
	}
	// If the action is unspecified, it means that the spec is effectively null.
	// Since it's null, don't do anything with it.
	if s.Action == QueryAction_UNSPECIFIED {
		return
	}

	ref.Action = ToV8Action(s.Action)

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
	policyIDs := sortx.Keys(d.Policies)
	for i := range policyIDs {
		id := policyIDs[i]
		spec := d.Policies[id]
		ref := &PolicyRef{}

		if spec != nil {
			ref.Action = ToV8Action(spec.Action)
		}

		if strings.HasPrefix(id, "//") && mrn.IsValid(id) {
			ref.Mrn = id
		} else {
			ref.Uid = id
		}

		policies[i] = ref
	}

	checks := make([]*explorer.Mquery, len(d.ScoringQueries))
	checkIDs := sortx.Keys(d.ScoringQueries)
	for i := range checkIDs {
		id := checkIDs[i]
		spec := d.ScoringQueries[id]
		ref := &explorer.Mquery{}
		spec.ApplyToV8(ref)

		if strings.HasPrefix(id, "//") && mrn.IsValid(id) {
			ref.Mrn = id
		} else {
			ref.Uid = id
		}

		checks[i] = ref
	}

	queries := make([]*explorer.Mquery, len(d.DataQueries))
	queryIDs := sortx.Keys(d.DataQueries)
	for i := range queryIDs {
		id := queryIDs[i]
		action := d.DataQueries[id]
		ref := &explorer.Mquery{}

		if action != QueryAction_UNSPECIFIED {
			ref.Action = ToV8Action(action)
		}

		if strings.HasPrefix(id, "//") && mrn.IsValid(id) {
			ref.Mrn = id
		} else {
			ref.Uid = id
		}

		queries[i] = ref
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

// DeprecatedV7_Add is a helper to add a policy and a bunch of queries to this bundle
func (bundle *PolicyBundleMap) DeprecatedV7_Add(policy *DeprecatedV7_Policy, queries map[string]*DeprecatedV7_Mquery) *PolicyBundleMap {
	var id string
	if policy.Mrn != "" {
		id = policy.Mrn
	} else {
		id = policy.Uid
	}

	bundle.Policies[id] = policy.ToV8()
	for k, v := range queries {
		bundle.Queries[k] = v.ToV8()
	}
	return bundle
}

// Conveting back to V7 structures
// -------------------------------

func ToV7Severity(i *explorer.Impact) *DeprecatedV7_SeverityValue {
	if i == nil {
		return nil
	}

	return &DeprecatedV7_SeverityValue{
		Value: int64(i.Value),
	}
}

func ToV7Mquery(x *explorer.Mquery) *DeprecatedV7_Mquery {
	if x == nil {
		return nil
	}

	return &DeprecatedV7_Mquery{
		Query:    x.Mql,
		CodeId:   x.CodeId,
		Checksum: x.Checksum,
		Mrn:      x.Mrn,
		Uid:      x.Uid,
		Type:     x.Type,
		Severity: ToV7Severity(x.Impact),
		Title:    x.Title,
		Docs:     ToV7MqueryDocs(x.Docs),
		Refs:     ToV7MqueryRefs(x.Refs),
		Tags:     x.Tags,
	}
}

func ToV7MqueryDocs(x *explorer.MqueryDocs) *DeprecatedV7_MqueryDocs {
	if x == nil {
		return nil
	}

	return &DeprecatedV7_MqueryDocs{
		Desc:        x.Desc,
		Audit:       x.Audit,
		Remediation: ToV7Remediation(x.Remediation),
	}
}

func ToV7Remediation(x *explorer.Remediation) string {
	if x == nil || len(x.Items) == 0 {
		return ""
	}

	return x.Items[0].Desc
}

func ToV7MqueryRefs(x []*explorer.MqueryRef) []*DeprecatedV7_MqueryRef {
	if x == nil {
		return nil
	}

	res := make([]*DeprecatedV7_MqueryRef, len(x))
	for i := range x {
		res[i] = ToV7MqueryRef(x[i])
	}
	return res
}

func ToV7MqueryRef(x *explorer.MqueryRef) *DeprecatedV7_MqueryRef {
	if x == nil {
		return nil
	}

	return &DeprecatedV7_MqueryRef{
		Title: x.Title,
		Url:   x.Url,
	}
}

func ToV7Filters(f *explorer.Filters) deprecatedV7_AssetFilters {
	if f == nil {
		return nil
	}

	res := map[string]*DeprecatedV7_Mquery{}
	for k, v := range f.Items {
		res[k] = ToV7Mquery(v)
	}

	return res
}

func ToV7SpecFilter(f *explorer.Filters, policyMrn string) *DeprecatedV7_Mquery {
	if f == nil || len(f.Items) == 0 {
		return nil
	}

	filters := []string{}
	for _, v := range f.Items {
		filters = append(filters, v.Mql)
	}

	res := &DeprecatedV7_Mquery{
		Query: strings.Join(filters, " || "),
	}
	_, err := res.RefreshAsAssetFilter(policyMrn)
	if err != nil {
		log.Error().Str("policy", policyMrn).Err(err).Msg("failed to convert filter to v7 for spec in policy")
	}

	return res
}

func ToV7ScoringSpec(action explorer.Action, impact *explorer.Impact) *DeprecatedV7_ScoringSpec {
	if action == explorer.Action_UNSPECIFIED {
		return nil
	}

	res := &DeprecatedV7_ScoringSpec{
		Action: ToV7Action(action),
	}

	if impact != nil && impact.Weight != -1 {
		res.Weight = uint32(impact.Weight)
	}

	return res
}

func ToV7Action(action explorer.Action) QueryAction {
	switch action {
	case explorer.Action_ACTIVATE:
		return QueryAction_ACTIVATE
	case explorer.Action_DEACTIVATE:
		return QueryAction_DEACTIVATE
	case explorer.Action_MODIFY:
		return QueryAction_MODIFY
	default:
		return QueryAction_UNSPECIFIED
	}
}

func ToV8Action(action QueryAction) explorer.Action {
	switch action {
	case QueryAction_ACTIVATE:
		return explorer.Action_ACTIVATE
	case QueryAction_DEACTIVATE:
		return explorer.Action_DEACTIVATE
	case QueryAction_MODIFY:
		return explorer.Action_MODIFY
	default:
		return explorer.Action_UNSPECIFIED
	}
}

func (x *PolicyGroup) ToV7(policyMrn string) *DeprecatedV7_PolicySpec {
	if x == nil {
		return nil
	}

	res := &DeprecatedV7_PolicySpec{
		Policies:       map[string]*DeprecatedV7_ScoringSpec{},
		ScoringQueries: map[string]*DeprecatedV7_ScoringSpec{},
		DataQueries:    map[string]QueryAction{},
	}

	for i := range x.Policies {
		p := x.Policies[i]
		if p.Mrn == "" {
			continue
		}
		res.Policies[p.Mrn] = ToV7ScoringSpec(p.Action, p.Impact)
	}
	for i := range x.Checks {
		check := x.Checks[i]
		if check.Mrn == "" {
			continue
		}
		res.ScoringQueries[check.Mrn] = ToV7ScoringSpec(check.Action, check.Impact)
	}
	for i := range x.Queries {
		query := x.Queries[i]
		if query.Mrn == "" {
			continue
		}
		res.DataQueries[query.Mrn] = ToV7Action(query.Action)
	}

	res.AssetFilter = ToV7SpecFilter(x.Filters, policyMrn)

	return res
}

func (x *Policy) FillV7() {
	if x == nil {
		return
	}

	x.AssetFilters = ToV7Filters(x.Filters)

	x.Specs = make([]*DeprecatedV7_PolicySpec, len(x.Groups))
	for i := range x.Groups {
		x.Specs[i] = x.Groups[i].ToV7(x.Mrn)
	}
}

func ToV7Authors(x []*explorer.Author) []*DeprecatedV7_Author {
	res := make([]*DeprecatedV7_Author, len(x))
	for i := range x {
		cur := x[i]
		res[i] = &DeprecatedV7_Author{
			Name:  cur.Name,
			Email: cur.Email,
		}
	}
	return res
}

func (x *Policy) ToV7() *DeprecatedV7_Policy {
	if x == nil {
		return nil
	}

	specs := make([]*DeprecatedV7_PolicySpec, len(x.Groups))
	for i := range x.Groups {
		specs[i] = x.Groups[i].ToV7(x.Mrn)
	}

	props := map[string]string{}
	for i := range x.Props {
		prop := x.Props[i]
		props[prop.Mrn] = ""
	}

	return &DeprecatedV7_Policy{
		Mrn:                    x.Mrn,
		Name:                   x.Name,
		Version:                x.Version,
		LocalContentChecksum:   x.LocalContentChecksum,
		GraphContentChecksum:   x.GraphContentChecksum,
		LocalExecutionChecksum: x.LocalExecutionChecksum,
		GraphExecutionChecksum: x.GraphExecutionChecksum,
		Specs:                  specs,
		AssetFilters:           ToV7Filters(x.Filters),
		OwnerMrn:               x.OwnerMrn,
		IsPublic:               false,
		ScoringSystem:          x.ScoringSystem,
		Authors:                ToV7Authors(x.Authors),
		Created:                x.Created,
		Modified:               x.Modified,
		Tags:                   x.Tags,
		Props:                  props,
		Uid:                    x.Uid,
		Docs:                   x.Docs,
		QueryCounts:            x.QueryCounts,
	}
}

func (x *Bundle) FillV7() {
	if x == nil {
		return
	}

	x.DeprecatedV7Policies = make([]*DeprecatedV7_Policy, len(x.Policies))
	for i := range x.Policies {
		x.DeprecatedV7Policies[i] = x.Policies[i].ToV7()
	}

	x.DeprecatedV7Queries = make([]*DeprecatedV7_Mquery, len(x.Queries))
	for i := range x.Queries {
		x.DeprecatedV7Queries[i] = ToV7Mquery(x.Queries[i])
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
