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
