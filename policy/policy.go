package policy

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/types"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

//go:generate protoc --proto_path=../:../cnquery:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnspec_policy.proto

// FIXME: DEPRECATED, remove in v9.0 vv
func (sv *DeprecatedV7_SeverityValue) UnmarshalJSON(data []byte) error {
	var sev int64

	if err := json.Unmarshal(data, &sev); err == nil {
		sv.Value = sev
	} else {
		v := &struct {
			Value int64 `json:"value"`
		}{}
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		sv.Value = v.Value
	}

	if sv.Value < 0 || sv.Value > 100 {
		return errors.New("severity must be between 0 and 100")
	}

	return nil
}

func (sv *DeprecatedV7_SeverityValue) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	return node.Decode(&sv.Value)
}

// ^^

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
	var found bool
	start := time.Now()
	ctx := context.Background()

	for time.Now().Sub(start) < timeout {
		res, err := resolver.GetScore(ctx, &EntityScoreReq{
			EntityMrn: entity,
			ScoreMrn:  scoringMrn,
		})
		if err != nil {
			return false, err
		}

		if res != nil && res.Score.ScoreCompletion == 100 && res.Score.DataCompletion == 100 {
			found = true
			log.Debug().
				Str("asset", entity).
				Str("type", res.Score.TypeLabel()).
				Int("value", int(res.Score.Value)).
				Int("score-completion", int(res.Score.ScoreCompletion)).
				Int("data-completion", int(res.Score.DataCompletion)).
				Int("data-total", int(res.Score.DataTotal)).
				Msg("waituntildone> got entity score")
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	return found, nil
}

// RefreshLocalAssetFilters looks through the local policy asset filters and rolls them up
func (p *Policy) RefreshLocalAssetFilters(lookupQueries map[string]*explorer.Mquery) {
	p.ComputedFilters = &explorer.Filters{
		Items: map[string]*explorer.Mquery{},
	}

	for i := range p.Groups {
		group := p.Groups[i]
		p.ComputedFilters.RegisterChild(group.Filters)

		for j := range group.Checks {
			check := group.Checks[j]
			if base, ok := lookupQueries[check.Mrn]; ok {
				check = check.Merge(base)
			}

			p.ComputedFilters.RegisterQuery(check, lookupQueries)
		}

		for j := range group.Queries {
			query := group.Queries[j]
			if base, ok := lookupQueries[query.Mrn]; ok {
				query = query.Merge(base)
			}

			p.ComputedFilters.RegisterQuery(query, lookupQueries)
		}
	}
}

func gatherLocalAssetFilters(ctx context.Context, groups []*PolicyGroup, lookupQueryByMrn func(ctx context.Context, mrn string) (*explorer.Mquery, error)) (*explorer.Filters, error) {
	filters := &explorer.Filters{
		Items: map[string]*explorer.Mquery{},
	}

	for _, group := range groups {
		filters.RegisterChild(group.Filters)

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

			if err := filters.RegisterQueryLookupFunc(ctx, check, lookupQueryByMrn); err != nil {
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

			if err := filters.RegisterQueryLookupFunc(ctx, query, lookupQueryByMrn); err != nil {
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
	recursive bool) ([]*explorer.Mquery, error) {
	filters := map[string]*explorer.Mquery{}

	localFilters, err := gatherLocalAssetFilters(ctx, p.Groups, getQuery)
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
	recursive bool, tracker map[string]*explorer.Mquery) error {
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
	getPolicy func(ctx context.Context, mrn string) (*Policy, error),
	getQuery func(ctx context.Context, mrn string) (*explorer.Mquery, error),
	bundle *PolicyBundleMap,
) error {
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

	// conditionals first: do we have local checksums set or not
	if p.LocalContentChecksum == "" || p.LocalExecutionChecksum == "" {
		return p.updateAllChecksums(ctx, getPolicy, getQuery, bundle)
	}

	// otherwise we have local checksums and only need to recompute the
	// graph checksums. This code is identical to the complete computation
	// but doesn't recompute any of the local checksums.

	graphExecutionChecksum := checksums.New
	graphContentChecksum := checksums.New

	var err error
	for i := range p.Groups {
		group := p.Groups[i]

		// POLICIES (must be sorted)
		policyMRNs := make([]string, len(group.Policies))
		i = 0
		for i := range group.Policies {
			policy := group.Policies[i]
			policyMRNs[i] = policy.Mrn
			i++
		}
		sort.Strings(policyMRNs)
		for _, policyMRN := range policyMRNs {
			p, ok := bundle.Policies[policyMRN]
			if !ok {
				p, err = getPolicy(ctx, policyMRN)
				if err != nil {
					return err
				}
			}

			if p.GraphContentChecksum == "" || p.GraphExecutionChecksum == "" {
				err = p.UpdateChecksums(ctx, getPolicy, getQuery, bundle)
				if err != nil {
					return err
				}
			}

			graphExecutionChecksum = graphExecutionChecksum.Add(p.GraphExecutionChecksum)
			graphContentChecksum = graphContentChecksum.Add(p.GraphContentChecksum)
		}
	}

	p.GraphExecutionChecksum = graphExecutionChecksum.Add(p.LocalExecutionChecksum).String()
	p.GraphContentChecksum = graphContentChecksum.Add(p.LocalContentChecksum).String()

	return nil
}

func (p *Policy) updateAllChecksums(ctx context.Context,
	getPolicy func(ctx context.Context, mrn string) (*Policy, error),
	getQuery func(ctx context.Context, mrn string) (*explorer.Mquery, error),
	bundle *PolicyBundleMap,
) error {
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

	// GROUPS
	for i := range p.Groups {
		group := p.Groups[i]

		// POLICIES (must be sorted)
		sort.Slice(group.Policies, func(i, j int) bool {
			return group.Policies[i].Mrn < group.Policies[j].Mrn
		})

		for i := range group.Policies {
			ref := group.Policies[i]

			p, err := getPolicy(ctx, ref.Mrn)
			if err != nil {
				return err
			}

			if p.GraphContentChecksum == "" || p.GraphExecutionChecksum == "" {
				return errors.New("failed to get checksums for dependent policy " + ref.Mrn)
			}

			executionChecksum = executionChecksum.Add(ref.Mrn)
			graphExecutionChecksum = graphExecutionChecksum.
				Add(p.GraphExecutionChecksum)
			graphContentChecksum = graphContentChecksum.
				Add(p.GraphContentChecksum)
		}

		// CHECKS (must be sorted)
		sort.Slice(group.Checks, func(i, j int) bool {
			return group.Checks[i].Mrn < group.Checks[j].Mrn
		})

		for i := range group.Checks {
			check := group.Checks[i]

			if base, ok := bundle.Queries[check.Mrn]; ok {
				check = check.Merge(base)
				if err := check.RefreshChecksum(ctx, getQuery); err != nil {
					return err
				}
			} else if check.Checksum == "" {
				if check.Mrn == "" {
					return errors.New("failed to get checksum for check " + check.Uid + ", MRN is empty")
				}
				if x, err := getQuery(ctx, check.Mrn); err == nil {
					check = x
				}
			}

			if check.Checksum == "" {
				return errors.New("failed to get checksum for check " + check.Mrn)
			}

			contentChecksum = contentChecksum.Add(check.Checksum)
			executionChecksum = executionChecksum.
				Add(check.CodeId).
				AddUint(check.Impact.Checksum())
		}

		// DATA (must be sorted)
		sort.Slice(group.Queries, func(i, j int) bool {
			return group.Queries[i].Mrn < group.Queries[j].Mrn
		})

		for i := range group.Queries {
			query := group.Queries[i]

			if base, ok := bundle.Queries[query.Mrn]; ok {
				query = query.Merge(base)
				if err := query.RefreshChecksum(ctx, getQuery); err != nil {
					return err
				}
			} else if query.Checksum == "" {
				if query.Mrn == "" {
					return errors.New("failed to get checksum for query " + query.Uid + ", MRN is empty")
				}
				if x, err := getQuery(ctx, query.Mrn); err == nil {
					query = x
				}
			}

			if query.Checksum == "" {
				return errors.New("failed to get checksum for query " + query.Mrn)
			}

			contentChecksum = contentChecksum.Add(query.Checksum)
			executionChecksum = executionChecksum.Add(query.CodeId)
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
					return errors.New("failed to get checksum for filter " + filter.Mrn)
				}
				if filter.CodeId == "" {
					return errors.New("failed to get code ID for filter " + filter.Mrn)
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
		}
	}

	p.LocalExecutionChecksum = executionChecksum.String()
	p.LocalContentChecksum = executionChecksum.AddUint(uint64(contentChecksum)).String()

	p.GraphExecutionChecksum = graphExecutionChecksum.Add(p.LocalExecutionChecksum).String()
	p.GraphContentChecksum = graphContentChecksum.Add(p.LocalContentChecksum).String()

	return nil
}

func checksumAddSpec(checksum checksums.Fast, spec *DeprecatedV7_ScoringSpec) checksums.Fast {
	checksum = checksum.AddUint((uint64(spec.Action) << 32) | (uint64(spec.ScoringSystem)))
	var weightIsPrecentage uint64
	if spec.WeightIsPercentage {
		weightIsPrecentage = 0x1 << 32
	}
	checksum = checksum.AddUint(weightIsPrecentage | uint64(spec.Weight))
	return checksum.Add(spec.Id)
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
