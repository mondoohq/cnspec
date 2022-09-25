package policy

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

//go:generate protoc --proto_path=../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. policy.proto

// WaitUntilDone for a score and an entity
func WaitUntilDone(resolver PolicyResolver, entity string, scoringMrn string, timeout time.Duration) (bool, error) {
	var found bool
	start := time.Now()
	ctx := context.Background()

	for time.Now().Sub(start) < timeout {
		res, err := resolver.GetScore(ctx, &EntityScoreRequest{
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
