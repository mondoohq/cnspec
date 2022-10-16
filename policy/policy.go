package policy

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/types"
	"google.golang.org/protobuf/proto"
)

//go:generate protoc --proto_path=../:../cnquery:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnspec_policy.proto

func (sv *SeverityValue) UnmarshalJSON(data []byte) error {
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
func (p *Policy) RefreshLocalAssetFilters() {
	p.AssetFilters = map[string]*Mquery{}

	for i := range p.Specs {
		spec := p.Specs[i]
		if spec.AssetFilter == nil {
			continue
		}

		filter := spec.AssetFilter
		filter.RefreshAsAssetFilter(p.Mrn)
		p.AssetFilters[filter.CodeId] = filter
	}
}

// ComputeAssetFilters of a given policy resolving them as you go
// recursive tells us if we want to call this function for all policy dependencies (costly; set to false by default)
func (p *Policy) ComputeAssetFilters(ctx context.Context, getPolicy func(ctx context.Context, mrn string) (*Policy, error), recursive bool) ([]*Mquery, error) {
	filters := map[string]*Mquery{}

	for i := range p.Specs {
		spec := p.Specs[i]

		// add asset filter of embeded policies
		if spec.AssetFilter != nil {
			filter := spec.AssetFilter
			filters[filter.Mrn] = filter
		}

		// add asset filter of child policies
		for mrn := range spec.Policies {
			if err := p.computeAssetFilters(ctx, mrn, getPolicy, recursive, filters); err != nil {
				return nil, err
			}
		}
	}

	res := make([]*Mquery, len(filters))
	var i int
	for _, v := range filters {
		res[i] = v
		i++
	}

	return res, nil
}

func (p *Policy) computeAssetFilters(ctx context.Context, policyMrn string, getPolicy func(ctx context.Context, mrn string) (*Policy, error), recursive bool, tracker map[string]*Mquery) error {
	child, err := getPolicy(ctx, policyMrn)
	if err != nil {
		return err
	}

	if recursive {
		childFilters, err := child.ComputeAssetFilters(ctx, getPolicy, recursive)
		if err != nil {
			return err
		}
		for i := range childFilters {
			c := childFilters[i]
			tracker[c.CodeId] = c
		}
	} else {
		for k, v := range child.AssetFilters {
			tracker[k] = v
		}
	}

	return nil
}

// MatchingAssetFilters will take the list of filters and only return the ones
// that are supported by the policy. if no matching field is found it will
// return an empty list
func MatchingAssetFilters(policyMrn string, assetFilters []*Mquery, p *Policy) ([]*Mquery, error) {
	res := []*Mquery{}
	for i := range assetFilters {
		cur := assetFilters[i]

		if _, ok := p.AssetFilters[cur.CodeId]; ok {
			curCopy := proto.Clone(cur).(*Mquery)
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

func getQueryNoop(ctx context.Context, mrn string) (*Mquery, error) {
	return nil, errors.New("query not found: " + mrn)
}

func (p *Policy) UpdateChecksums(ctx context.Context,
	getPolicy func(ctx context.Context, mrn string) (*Policy, error),
	getQuery func(ctx context.Context, mrn string) (*Mquery, error),
	bundle *PolicyBundleMap,
) error {
	// simplify the access if we don't have a bundle
	if bundle == nil {
		bundle = &PolicyBundleMap{
			Queries: map[string]*Mquery{},
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
	for i := range p.Specs {
		spec := p.Specs[i]

		// POLICIES (must be sorted)
		policyMRNs := make([]string, len(spec.Policies))
		i = 0
		for k := range spec.Policies {
			policyMRNs[i] = k
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
	getQuery func(ctx context.Context, mrn string) (*Mquery, error),
	bundle *PolicyBundleMap,
) error {
	log.Trace().Str("policy", p.Mrn).Msg("update policy checksum")
	p.LocalContentChecksum = ""
	p.LocalExecutionChecksum = ""

	// Note: this relies on the fact that the bundle was compiled before
	// We include the hash for scoring queries and data queries here since changes in the
	// queries are otherwise not reflected in the polciy checksum. While the policy checksum
	// tracks all changes within the policy object itself (like spec and assigned queries),
	// it does not track changes in the assigned queries itself.
	//
	// This has a lot of side-effects for shared queries between different policies
	// e.g. Policy1 refs Query1 and Policy2 refs Query1. Policy1 is already uploaded with
	// Query1. Now, we upload the policy bundle with both Policies and an updated Query1
	// If Query1 change is not included in the checksum, the generated score object for Policy1
	// is wrong after Policy2 is stored (since the query is globally changed). Therefore we need to
	// update the policy when the policy or an underlying query has been changed

	var err error
	var i int

	executionChecksum := checksums.New
	contentChecksum := checksums.New
	graphExecutionChecksum := checksums.New
	graphContentChecksum := checksums.New

	// content fields in the policy
	contentChecksum = contentChecksum.Add(p.Mrn).Add(p.Name).Add(p.Version).Add(p.OwnerMrn)
	if p.IsPublic {
		contentChecksum = contentChecksum.AddUint(1)
	} else {
		contentChecksum = contentChecksum.AddUint(0)
	}
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
	if p.ScoringSystem == ScoringSystem_SCORING_UNSPECIFIED {
		p.ScoringSystem = ScoringSystem_AVERAGE
	}
	executionChecksum = executionChecksum.Add(p.ScoringSystem.String())

	// PROPS (must be sorted)
	queryIDs := make([]string, len(p.Props))
	i = 0
	for k := range p.Props {
		queryIDs[i] = k
		i++
	}
	sort.Strings(queryIDs)
	for _, queryID := range queryIDs {
		q, ok := bundle.Props[queryID]
		if !ok {
			q, err = getQuery(ctx, queryID)
			if err != nil {
				return err
			}
		}
		executionChecksum = executionChecksum.Add(q.Checksum)
		executionChecksum = executionChecksum.Add(p.Props[queryID])
	}

	// SPECS
	for i := range p.Specs {
		spec := p.Specs[i]

		// POLICIES (must be sorted)
		policyMRNs := make([]string, len(spec.Policies))
		idx := 0
		for k := range spec.Policies {
			policyMRNs[idx] = k
			idx++
		}
		sort.Strings(policyMRNs)
		for _, policyMRN := range policyMRNs {
			executionChecksum = executionChecksum.Add(policyMRN)
			if spec := spec.Policies[policyMRN]; spec != nil {
				executionChecksum = checksumAddSpec(executionChecksum, spec)
			}

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

		// SCORING (must be sorted)
		queryIDs = make([]string, len(spec.ScoringQueries))
		idx = 0
		for k := range spec.ScoringQueries {
			queryIDs[idx] = k
			idx++
		}
		sort.Strings(queryIDs)
		for _, queryID := range queryIDs {
			q, ok := bundle.Queries[queryID]
			if !ok {
				q, err = getQuery(ctx, queryID)
				if err != nil {
					return err
				}
			}

			// we use the checksum for doc, tag and ref changes
			contentChecksum = contentChecksum.Add(q.Checksum)
			executionChecksum = executionChecksum.Add(q.CodeId)

			if spec := spec.ScoringQueries[queryID]; spec != nil {
				executionChecksum = checksumAddSpec(executionChecksum, spec)
			}
		}

		// DATA (must be sorted)
		queryIDs = make([]string, len(spec.DataQueries))
		idx = 0
		for k := range spec.DataQueries {
			queryIDs[idx] = k
			idx++
		}
		sort.Strings(queryIDs)
		for i := range queryIDs {
			queryID := queryIDs[i]
			q, ok := bundle.Queries[queryID]
			if !ok {
				q, err = getQuery(ctx, queryID)
				if err != nil {
					return err
				}
			}

			// we use the checksum for doc, tag and ref changes
			contentChecksum = contentChecksum.Add(q.Checksum)
			executionChecksum = executionChecksum.Add(q.CodeId)

			if action, ok := spec.DataQueries[queryID]; ok {
				executionChecksum = executionChecksum.AddUint(uint64(action))
			}
		}

		// ASSET FILTER
		q := spec.AssetFilter
		if q != nil {
			contentChecksum = contentChecksum.Add(q.Checksum)
			executionChecksum = executionChecksum.Add(q.CodeId)
		}

		// REMAINING FIELDS
		executionChecksum = executionChecksum.
			AddUint(uint64(spec.StartDate)).
			AddUint(uint64(spec.EndDate))

		// other content fields
		contentChecksum = contentChecksum.
			AddUint(uint64(spec.ReminderDate)).
			AddUint(uint64(spec.Created)).
			AddUint(uint64(spec.Modified)).
			Add(spec.Title)
		if spec.Docs != nil {
			contentChecksum = contentChecksum.
				Add(spec.Docs.Desc)
		}
	}

	p.LocalExecutionChecksum = executionChecksum.String()
	p.LocalContentChecksum = executionChecksum.AddUint(uint64(contentChecksum)).String()

	p.GraphExecutionChecksum = graphExecutionChecksum.Add(p.LocalExecutionChecksum).String()
	p.GraphContentChecksum = graphContentChecksum.Add(p.LocalContentChecksum).String()

	return nil
}

func checksumAddSpec(checksum checksums.Fast, spec *ScoringSpec) checksums.Fast {
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
	for i := range p.Specs {
		spec := p.Specs[i]
		for k := range spec.Policies {
			mrns[k] = struct{}{}
		}
	}

	return mrns
}

// RefreshMRN computes a MRN from the UID or validates the existing MRN.
// Both of these need to fit the ownerMRN. It also removes the UID.
func (p *Policy) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, p.Mrn, "policies", p.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", p.Uid).Msg("failed to refresh mrn")
		return errors.Wrap(err, "failed to refresh mrn for query "+p.Name)
	}

	p.Mrn = nu
	p.Uid = ""
	return nil
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

func (s *ScoringSystem) UnmarshalJSON(data []byte) error {
	// check if we have a number
	var code int32
	err := json.Unmarshal(data, &code)
	if err == nil {
		*s = ScoringSystem(code)
	} else {
		var name string
		_ = json.Unmarshal(data, &name)

		switch name {
		case "highest impact":
			*s = ScoringSystem_WORST
		case "weighted":
			*s = ScoringSystem_WEIGHTED
		case "average", "":
			*s = ScoringSystem_AVERAGE
		default:
			return errors.New("unknown scoring system: " + string(data))
		}
	}
	return nil
}
