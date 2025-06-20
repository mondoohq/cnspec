// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/checksums"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/mrn"
	"go.mondoo.com/cnquery/v11/types"
	"go.mondoo.com/cnquery/v11/utils/sortx"
	"google.golang.org/protobuf/proto"
)

//go:generate protoc --proto_path=../:../cnquery:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. --go-vtproto_out=. --go-vtproto_opt=paths=source_relative --go-vtproto_opt=features=marshal+unmarshal+size+clone cnspec_policy.proto

type dataQueryInfo struct {
	Type   types.Type `json:"type,omitempty"`
	Notify []string   `json:"notify,omitempty"`
}

// MarshalJSON generates escapes the \u0000 string for postgres
// Otherwise we are not able to store the compile code as json blob in pg since
// llx and type use \x00 or \u0000. This is not allowed in Postgres json blobs
// see https://www.postgresql.org/docs/9.4/release-9-4-1.html
// TODO: this is a workaround and we should not store this as json in pg in the first place
func (dqi *DataQueryInfo) MarshalJSON() ([]byte, error) {
	jsonQueryInfo := dataQueryInfo{
		Type:   types.Type(dqi.Type),
		Notify: dqi.Notify,
	}

	return json.Marshal(jsonQueryInfo)
}

// UnmarshalJSON reverts MarshalJSON data arrays to its base type.
func (dqi *DataQueryInfo) UnmarshalJSON(data []byte) error {
	res := dataQueryInfo{}
	err := json.Unmarshal(data, &res)
	if err != nil {
		return err
	}
	dqi.Notify = res.Notify
	dqi.Type = string(res.Type)
	return nil
}

// WaitUntilDone for a score and an entity
func WaitUntilDone(resolver PolicyResolver, entity string, scoringMrn string, timeout time.Duration) (bool, error) {
	start := time.Now()
	ctx := context.Background()

	for time.Since(start) < timeout {
		res, err := resolver.GetScore(ctx, &EntityScoreReq{
			EntityMrn: entity,
			ScoreMrn:  scoringMrn,
		})
		if err != nil {
			return false, err
		}

		if res != nil && res.Score.ScoreCompletion == 100 && res.Score.DataCompletion == 100 {
			log.Debug().
				Str("asset", entity).
				Str("type", res.Score.TypeLabel()).
				Int("value", int(res.Score.Value)).
				Int("score-completion", int(res.Score.ScoreCompletion)).
				Int("data-completion", int(res.Score.DataCompletion)).
				Int("data-total", int(res.Score.DataTotal)).
				Msg("waituntildone> got entity score")
			return true, nil
		}

		time.Sleep(50 * time.Millisecond)
	}

	return false, nil
}

func cannotLookupFilters(ctx context.Context, mrn string) (*explorer.Mquery, error) {
	return nil, errors.New("cannot look up filters for mrn=" + mrn)
}

// Gather only the local asset filters, which means we don't descend into
// dependent queries. Due to variants and referenced remote queries we may
// still need to look up queries to get to their filters.
func gatherLocalAssetFilters(ctx context.Context, policy *Policy, lookupQueryByMrn func(ctx context.Context, mrn string) (*explorer.Mquery, error)) (*explorer.Filters, error) {
	groups := policy.Groups

	filters := &explorer.Filters{
		Items: map[string]*explorer.Mquery{},
	}

	for _, group := range groups {
		filters.AddFilters(group.Filters)

		for k, m := range group.Filters.GetItems() {
			filters.Items[k] = m
		}

		for _, check := range group.Checks {
			base, _ := lookupQueryByMrn(ctx, check.Mrn) //nolint: errcheck
			// The implementations of the getQuery interface all return random error messages
			// that is not well defined.

			if base != nil {
				check = check.Merge(base)
			}

			if err := filters.AddQueryFiltersFn(ctx, check, lookupQueryByMrn); err != nil {
				return nil, err
			}
		}

		for _, query := range group.Queries {
			base, _ := lookupQueryByMrn(ctx, query.Mrn) //nolint: errcheck
			// The implementations of the getQuery interface all return random error messages
			// that is not well defined.

			if base != nil {
				query = query.Merge(base)
			}

			if err := filters.AddQueryFiltersFn(ctx, query, lookupQueryByMrn); err != nil {
				return nil, err
			}
		}
	}

	for i := range policy.RiskFactors {
		rf := policy.RiskFactors[i]
		filters.AddFilters(rf.Filters)
		for j := range rf.Checks {
			if err := filters.AddQueryFiltersFn(ctx, rf.Checks[j], cannotLookupFilters); err != nil {
				return nil, err
			}
		}
	}

	return filters, nil
}

// ComputeAssetFilters of a given policy resolving them as you go
// recursive tells us if we want to call this function for all policy dependencies (costly; set to false by default)
func (p *Policy) ComputeAssetFilters(ctx context.Context,
	getPolicy func(ctx context.Context, mrn string) (*Policy, error),
	getQuery func(ctx context.Context, mrn string) (*explorer.Mquery, error),
	recursive bool,
) ([]*explorer.Mquery, error) {
	filters := map[string]*explorer.Mquery{}

	localFilters, err := gatherLocalAssetFilters(ctx, p, getQuery)
	if err != nil {
		return nil, err
	}
	for k, m := range localFilters.GetItems() {
		filters[k] = m
	}

	for i := range p.Groups {
		group := p.Groups[i]
		// add asset filter of child policies
		for i := range group.Policies {
			if err := p.computeAssetFilters(ctx, group.Policies[i].Mrn, getPolicy, getQuery, recursive, filters); err != nil {
				return nil, err
			}
		}
	}

	for i := range p.RiskFactors {
		rf := p.RiskFactors[i]
		if rf.Filters != nil {
			for i := range rf.Filters.Items {
				filters[i] = rf.Filters.Items[i]
			}
		}
	}

	res := make([]*explorer.Mquery, len(filters))
	var i int
	for _, v := range filters {
		res[i] = v
		i++
	}

	return res, nil
}

func (p *Policy) computeAssetFilters(ctx context.Context, policyMrn string,
	getPolicy func(ctx context.Context, mrn string) (*Policy, error),
	getQuery func(ctx context.Context, mrn string) (*explorer.Mquery, error),
	recursive bool, tracker map[string]*explorer.Mquery,
) error {
	child, err := getPolicy(ctx, policyMrn)
	if err != nil {
		return err
	}

	if recursive {
		childFilters, err := child.ComputeAssetFilters(ctx, getPolicy, getQuery, recursive)
		if err != nil {
			return err
		}
		for i := range childFilters {
			c := childFilters[i]
			tracker[c.CodeId] = c
		}
	} else if child.ComputedFilters != nil {
		for i := range child.ComputedFilters.Items {
			filter := child.ComputedFilters.Items[i]
			tracker[filter.CodeId] = filter
		}
	}

	return nil
}

// MatchingAssetFilters will take the list of filters and only return the ones
// that are supported by the policy. if no matching field is found it will
// return an empty list
func MatchingAssetFilters(policyMrn string, assetFilters []*explorer.Mquery, p *Policy) ([]*explorer.Mquery, error) {
	if p.ComputedFilters == nil || len(p.ComputedFilters.Items) == 0 {
		return nil, nil
	}

	policyFilters := map[string]struct{}{}
	for i := range p.ComputedFilters.Items {
		policyFilters[p.ComputedFilters.Items[i].CodeId] = struct{}{}
	}

	res := []*explorer.Mquery{}
	for i := range assetFilters {
		cur := assetFilters[i]

		if _, ok := policyFilters[cur.CodeId]; ok {
			curCopy := proto.Clone(cur).(*explorer.Mquery)
			curCopy.Mrn = policyMrn + "/assetfilter/" + cur.CodeId
			curCopy.Title = curCopy.Query
			res = append(res, curCopy)
		}
	}
	return res, nil
}

func getPolicyNoop(ctx context.Context, mrn string) (*Policy, error) {
	return nil, errors.New("policy not found: " + mrn)
}

func getQueryNoop(ctx context.Context, mrn string) (*explorer.Mquery, error) {
	return nil, errors.New("query not found: " + mrn)
}

func (p *Policy) UpdateChecksums(ctx context.Context,
	now time.Time,
	getPolicy func(ctx context.Context, mrn string) (*Policy, error),
	getQuery func(ctx context.Context, mrn string) (*explorer.Mquery, error),
	bundle *PolicyBundleMap,
	conf mqlc.CompilerConfig,
) (*time.Time, error) {
	// simplify the access if we don't have a bundle
	if bundle == nil {
		bundle = &PolicyBundleMap{
			Queries: map[string]*explorer.Mquery{},
		}
	}

	if getPolicy == nil {
		getPolicy = getPolicyNoop
	}

	if getQuery == nil {
		getQuery = getQueryNoop
	}

	// if we have local checksums set, we can take an optimized route;
	// if not, we have to update all checksums
	if p.LocalContentChecksum == "" || p.LocalExecutionChecksum == "" {
		return p.updateAllChecksums(ctx, now, getPolicy, getQuery, bundle, conf)
	}

	// otherwise we have local checksums and only need to recompute the
	// graph checksums. This code is identical to the complete computation
	// but doesn't recompute any of the local checksums.

	graphExecutionChecksum := checksums.New
	graphContentChecksum := checksums.New
	recalculateAt := p.recalculateAt(now)
	var err error
	for i := range p.Groups {
		group := p.Groups[i]

		// POLICIES (must be sorted)
		policyMRNs := make([]string, len(group.Policies))
		for i := range group.Policies {
			policyMRNs[i] = group.Policies[i].Mrn
		}
		sort.Strings(policyMRNs)
		for _, policyMRN := range policyMRNs {
			p, ok := bundle.Policies[policyMRN]
			if !ok {
				p, err = getPolicy(ctx, policyMRN)
				if err != nil {
					return recalculateAt, err
				}
				ra := p.recalculateAt(now)
				if ra != nil {
					if recalculateAt == nil || ra.Before(*recalculateAt) {
						recalculateAt = ra
					}
				}
			}

			if p.GraphContentChecksum == "" || p.GraphExecutionChecksum == "" {
				_, err = p.UpdateChecksums(ctx, now, getPolicy, getQuery, bundle, conf)
				if err != nil {
					return recalculateAt, err
				}
			}

			graphExecutionChecksum = graphExecutionChecksum.Add(p.GraphExecutionChecksum)
			graphContentChecksum = graphContentChecksum.Add(p.GraphContentChecksum)
		}
	}

	p.GraphExecutionChecksum = graphExecutionChecksum.Add(p.LocalExecutionChecksum).String()
	p.GraphContentChecksum = graphContentChecksum.Add(p.LocalContentChecksum).String()

	return recalculateAt, nil
}

func (p *Policy) recalculateAt(now time.Time) *time.Time {
	var timeToRecalculate time.Time

	updateTimeToRecalculate := func(tUnix int64) {
		if tUnix == 0 {
			return
		}
		t := time.Unix(tUnix, 0)
		if !t.Before(now) && (timeToRecalculate.IsZero() || t.Before(timeToRecalculate)) {
			timeToRecalculate = t
		}
	}

	for _, g := range p.Groups {
		updateTimeToRecalculate(g.StartDate)
		updateTimeToRecalculate(g.EndDate)
	}

	if timeToRecalculate.IsZero() {
		return nil
	}
	return &timeToRecalculate
}

func (p *Policy) updateAllChecksums(ctx context.Context,
	now time.Time,
	getPolicy func(ctx context.Context, mrn string) (*Policy, error),
	getQuery func(ctx context.Context, mrn string) (*explorer.Mquery, error),
	bundle *PolicyBundleMap,
	conf mqlc.CompilerConfig,
) (*time.Time, error) {
	log.Trace().Str("policy", p.Mrn).Msg("update policy checksum")
	p.LocalContentChecksum = ""
	p.LocalExecutionChecksum = ""

	// Note: this relies on the fact that the bundle was compiled before
	// We include the hash for scoring queries and data queries here since changes in the
	// queries are otherwise not reflected in the policy checksum. While the policy checksum
	// tracks all changes within the policy object itself (like spec and assigned queries),
	// it does not track changes in the assigned queries itself.
	//
	// This has a lot of side-effects for shared queries between different policies
	// e.g. Policy1 refs Query1 and Policy2 refs Query1. Policy1 is already uploaded with
	// Query1. Now, we upload the policy bundle with both Policies and an updated Query1
	// If Query1 change is not included in the checksum, the generated score object for Policy1
	// is wrong after Policy2 is stored (since the query is globally changed). Therefore we need to
	// update the policy when the policy or an underlying query has been changed

	var i int

	executionChecksum := checksums.New
	contentChecksum := checksums.New
	graphExecutionChecksum := checksums.New
	graphContentChecksum := checksums.New

	// content fields in the policy
	contentChecksum = contentChecksum.Add(p.Mrn).Add(p.Name).Add(p.Version).Add(p.OwnerMrn)
	for i := range p.Authors {
		author := p.Authors[i]
		contentChecksum = contentChecksum.Add(author.Email).Add(author.Name)
	}
	contentChecksum = contentChecksum.AddUint(uint64(p.Created)).AddUint(uint64(p.Modified))

	if p.Docs != nil {
		contentChecksum = contentChecksum.Add(p.Docs.Desc)
	}

	// Special handling for asset MRNs: While for most policies the MRN is
	// important, for assets that's not the case. We can safely ignore it for
	// the sake of the execution checksum. This also helps to indicate where
	// policies overlap.
	if x, _ := mrn.GetResource(p.Mrn, MRN_RESOURCE_ASSET); x != "" {
		executionChecksum = executionChecksum.Add("root")
	} else {
		executionChecksum = executionChecksum.Add(p.Mrn)
	}

	// tags
	arr := make([]string, len(p.Tags))
	i = 0
	for k := range p.Tags {
		arr[i] = k
		i++
	}
	sort.Strings(arr)
	for _, k := range arr {
		contentChecksum = contentChecksum.Add(k).Add(p.Tags[k])
	}

	// execution fields in policy
	if p.ScoringSystem == explorer.ScoringSystem_SCORING_UNSPECIFIED {
		p.ScoringSystem = explorer.ScoringSystem_AVERAGE
	}
	executionChecksum = executionChecksum.AddUint(uint64(p.ScoringSystem))

	// PROPS (must be sorted)
	sort.Slice(p.Props, func(i, j int) bool {
		return p.Props[i].Mrn < p.Props[j].Mrn
	})
	for i := range p.Props {
		executionChecksum = executionChecksum.Add(p.Props[i].Checksum)
	}

	recalculateAt := p.recalculateAt(now)
	if recalculateAt != nil {
		executionChecksum = executionChecksum.AddUint(uint64(recalculateAt.Unix()))
	} else {
		executionChecksum = executionChecksum.AddUint(0)
	}

	// GROUPS
	for i := range p.Groups {
		group := p.Groups[i]

		contentChecksum = contentChecksum.Add(group.Uid)

		// POLICIES (must be sorted)
		sort.Slice(group.Policies, func(i, j int) bool {
			return group.Policies[i].Mrn < group.Policies[j].Mrn
		})

		executionChecksum = executionChecksum.AddUint(uint64(group.Type))
		executionChecksum = executionChecksum.AddUint(uint64(group.ReviewStatus))
		for i := range group.Policies {
			ref := group.Policies[i]

			p, err := getPolicy(ctx, ref.Mrn)
			if err != nil {
				return recalculateAt, err
			}

			if p.GraphContentChecksum == "" || p.GraphExecutionChecksum == "" {
				return recalculateAt, errors.New("failed to get checksums for dependent policy " + ref.Mrn)
			}
			ra := p.recalculateAt(now)
			if ra != nil {
				if recalculateAt == nil || ra.Before(*recalculateAt) {
					recalculateAt = ra
				}
			}

			executionChecksum = executionChecksum.Add(ref.Mrn)
			executionChecksum = executionChecksum.AddUint(uint64(ref.Action))
			graphExecutionChecksum = graphExecutionChecksum.
				Add(p.GraphExecutionChecksum)
			graphContentChecksum = graphContentChecksum.
				Add(p.GraphContentChecksum)
		}

		// CHECKS (must be sorted)
		// copy checks to keep the original order and only sort it for the purpose of checksum generation
		checks := make([]*explorer.Mquery, len(group.Checks))
		copy(checks, group.Checks)
		sort.Slice(checks, func(i, j int) bool {
			return checks[i].Mrn < checks[j].Mrn
		})

		for i := range checks {
			check := checks[i]

			if base, ok := bundle.Queries[check.Mrn]; ok {
				check = check.Merge(base)
				if err := check.RefreshChecksum(ctx, conf, getQuery); err != nil {
					return recalculateAt, err
				}
			} else if check.Checksum == "" {
				if check.Mrn == "" {
					return recalculateAt, errors.New("failed to get checksum for check " + check.Uid + ", MRN is empty")
				}
				if x, err := getQuery(ctx, check.Mrn); err == nil {
					check = check.Merge(x)
					if err := check.RefreshChecksum(ctx, conf, getQuery); err != nil {
						return recalculateAt, err
					}
				}
			}

			if check.Checksum == "" {
				return recalculateAt, errors.New("failed to get checksum for check " + check.Mrn)
			}

			contentChecksum = contentChecksum.Add(check.Checksum)

			var err error
			executionChecksum, err = variantsExecutionChecksum(check, executionChecksum, true, getQuery)
			if err != nil {
				return recalculateAt, err
			}

			for _, p := range check.Props {
				executionChecksum = executionChecksum.Add(p.Checksum)
			}
		}

		// DATA (must be sorted)
		// copy checks to keep the original order and only sort it for the purpose of checksum generation
		queries := make([]*explorer.Mquery, len(group.Queries))
		copy(queries, group.Queries)
		sort.Slice(queries, func(i, j int) bool {
			return queries[i].Mrn < queries[j].Mrn
		})

		for i := range queries {
			query := queries[i]

			if base, ok := bundle.Queries[query.Mrn]; ok {
				query = query.Merge(base)
				if err := query.RefreshChecksum(ctx, conf, getQuery); err != nil {
					return recalculateAt, err
				}
			} else if query.Checksum == "" {
				if query.Mrn == "" {
					return recalculateAt, errors.New("failed to get checksum for query " + query.Uid + ", MRN is empty")
				}
				if x, err := getQuery(ctx, query.Mrn); err == nil {
					query = query.Merge(x)
					if err := query.RefreshChecksum(ctx, conf, getQuery); err != nil {
						return recalculateAt, err
					}
				}
			}

			if query.Checksum == "" {
				return recalculateAt, errors.New("failed to get checksum for query " + query.Mrn)
			}

			contentChecksum = contentChecksum.Add(query.Checksum)

			var err error
			executionChecksum, err = variantsExecutionChecksum(query, executionChecksum, false, getQuery)
			if err != nil {
				return recalculateAt, err
			}

			for _, p := range query.Props {
				executionChecksum = executionChecksum.Add(p.Checksum)
			}
		}

		// FILTERs (also sorted)
		if group.Filters != nil {
			keys := make([]string, len(group.Filters.Items))
			i := 0
			for k := range group.Filters.Items {
				keys[i] = k
				i++
			}
			sort.Strings(keys)

			for i := range keys {
				key := keys[i]
				filter := group.Filters.Items[key]
				if filter.Checksum == "" {
					return recalculateAt, errors.New("failed to get checksum for filter " + filter.Mrn)
				}
				if filter.CodeId == "" {
					return recalculateAt, errors.New("failed to get code ID for filter " + filter.Mrn)
				}

				contentChecksum = contentChecksum.Add(filter.Checksum)
				executionChecksum = executionChecksum.Add(filter.CodeId)
			}
		}

		// REMAINING FIELDS
		executionChecksum = executionChecksum.
			AddUint(uint64(group.StartDate)).
			AddUint(uint64(group.EndDate))

		// other content fields
		contentChecksum = contentChecksum.
			AddUint(uint64(group.ReminderDate)).
			AddUint(uint64(group.Created)).
			AddUint(uint64(group.Modified)).
			Add(group.Title)
		if group.Docs != nil {
			contentChecksum = contentChecksum.
				Add(group.Docs.Desc)
			contentChecksum = contentChecksum.Add(group.Docs.Justification)
		}
	}

	// RISKS
	riskIdx := make(map[string]*RiskFactor, len(p.RiskFactors))
	for i := range p.RiskFactors {
		cur := p.RiskFactors[i]
		riskIdx[cur.Mrn] = cur
	}

	sortedRiskMRNs := sortx.Keys(riskIdx)
	for _, riskMRN := range sortedRiskMRNs {
		esum, csum, err := riskIdx[riskMRN].RefreshChecksum(ctx, conf)
		if err != nil {
			return recalculateAt, err
		}
		executionChecksum = executionChecksum.AddUint(uint64(esum))
		contentChecksum = contentChecksum.AddUint(uint64(csum))
	}

	p.LocalExecutionChecksum = executionChecksum.String()
	p.LocalContentChecksum = executionChecksum.AddUint(uint64(contentChecksum)).String()

	p.GraphExecutionChecksum = graphExecutionChecksum.Add(p.LocalExecutionChecksum).String()
	p.GraphContentChecksum = graphContentChecksum.Add(p.LocalContentChecksum).String()

	return recalculateAt, nil
}

func (p *Policy) InvalidateGraphChecksums() {
	p.GraphContentChecksum = ""
	p.GraphExecutionChecksum = ""
}

func (p *Policy) InvalidateLocalChecksums() {
	p.LocalContentChecksum = ""
	p.LocalExecutionChecksum = ""
}

func (p *Policy) InvalidateExecutionChecksums() {
	p.LocalExecutionChecksum = ""
	p.GraphExecutionChecksum = ""
}

func (p *Policy) InvalidateAllChecksums() {
	p.LocalContentChecksum = ""
	p.LocalExecutionChecksum = ""
	p.GraphContentChecksum = ""
	p.GraphExecutionChecksum = ""
}

// DependentPolicyMrns lists all policies found across all specs
func (p *Policy) DependentPolicyMrns() map[string]struct{} {
	mrns := map[string]struct{}{}
	for i := range p.Groups {
		group := p.Groups[i]
		for k := range group.Policies {
			mrns[group.Policies[k].Mrn] = struct{}{}
		}
	}

	return mrns
}

// RefreshMRN computes a MRN from the UID or validates the existing MRN.
// Both of these need to fit the ownerMRN. It also removes the UID.
func (p *Policy) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, p.Mrn, "policies", p.Uid)
	if err != nil {
		log.Debug().Err(err).Str("owner", ownerMRN).Str("uid", p.Uid).Msg("failed to refresh mrn")
		return errors.Wrap(err, "failed to refresh mrn for policy "+p.Name+" "+p.Uid)
	}

	p.Mrn = nu
	p.Uid = ""
	return nil
}

// RefreshMRN computes a MRN from the UID or validates the existing MRN.
// Both of these need to fit the ownerMRN. It also removes the UID.
func (p *PolicyRef) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, p.Mrn, "policies", p.Uid)
	if err != nil {
		log.Debug().Err(err).Str("owner", ownerMRN).Str("uid", p.Uid).Msg("failed to refresh mrn")
		return errors.Wrap(err, "failed to refresh mrn for policy reference "+p.Uid)
	}

	p.Mrn = nu
	p.Uid = ""
	return nil
}

func (p *PolicyRef) RefreshChecksum() {
	c := checksums.New.
		Add(p.Mrn).
		AddUint(uint64(p.Action)).
		AddUint(p.Impact.Checksum())

	p.Checksum = c.String()
}

func IsPolicyMrn(candidate string) error {
	policyID, err := mrn.GetResource(candidate, MRN_RESOURCE_POLICY)
	if err != nil {
		return errors.New("failed to parse policy MRN " + candidate)
	}
	if policyID == "" {
		return errors.New("policy MRN is invalid, no policy ID in " + candidate)
	}
	return nil
}

func (s *GroupType) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*s = GroupType_UNCATEGORIZED
		return nil
	}

	type tmp GroupType
	err := json.Unmarshal(data, (*tmp)(s))
	if err == nil {
		return nil
	}

	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return errors.New("failed to unmarshal group type: " + string(data))
	}

	v := strings.ToUpper(string(str))
	if x, ok := GroupType_value[v]; ok {
		*s = GroupType(x)
		return nil
	}

	return errors.New("failed to unmarshal group type: " + str)
}

func (a *Migration_Action) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*a = Migration_UNSPECIFIED
		return nil
	}

	v := strings.ToUpper(strings.Trim(string(data), "\""))
	switch v {
	case "CREATE":
		*a = Migration_CREATE
	case "REMOVE":
		*a = Migration_REMOVE
	case "MODIFY":
		*a = Migration_MODIFY
	default:
		type tmp Migration_Action
		err := json.Unmarshal(data, (*tmp)(a))
		if err != nil {
			return errors.New("failed to unmarshal '" + string(data) + "' into migration action")
		}
	}

	return nil
}

func variantsExecutionChecksum(q *explorer.Mquery, c checksums.Fast, includeImpact bool, getQuery func(ctx context.Context, mrn string) (*explorer.Mquery, error)) (checksums.Fast, error) {
	// This code assumes there are no cycles in the variant graph.
	c = c.
		Add(q.CodeId)
	if includeImpact {
		c = c.AddUint(q.Impact.Checksum())
	}

	for _, ref := range q.Variants {
		if v, err := getQuery(context.Background(), ref.Mrn); err == nil {
			c, err = variantsExecutionChecksum(v, c, includeImpact, getQuery)
			if err != nil {
				return 0, err
			}
		} else {
			return 0, errors.New("cannot find dependent composed query '" + ref.Mrn + "'")
		}
	}
	return c, nil
}
