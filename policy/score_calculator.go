// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"strconv"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/v9/explorer"
	"google.golang.org/protobuf/proto"
)

// ScoreCalculator interface for calculating scores
type ScoreCalculator interface {
	Add(score *Score)
	Calculate() *Score
	Init()
}

type averageScoreCalculator struct {
	value                 uint32
	weight                uint32
	scoreTotal            uint32
	scoreCompletion       uint32
	scoreCnt              uint32
	dataTotal             uint32
	dataCompletion        uint32
	hasResults            bool
	hasErrors             bool
	featureFlagFailErrors bool
}

func (c *averageScoreCalculator) Init() {
	c.value = 0
	c.weight = 0
	c.scoreTotal = 0
	c.scoreCompletion = 0
	c.scoreCnt = 0
	c.dataTotal = 0
	c.dataCompletion = 0
	c.hasResults = false
	c.hasErrors = false
}

func AddSpecdScore(calculator ScoreCalculator, s *Score, found bool, impact *explorer.Impact) {
	if !found {
		calculator.Add(&Score{
			ScoreCompletion: 0,
			DataCompletion:  0,
		})
		return
	}

	score := proto.Clone(s).(*Score)
	if impact != nil && impact.Value != nil {
		floor := 100 - uint32(impact.Value.Value)
		if floor > score.Value {
			score.Value = floor
		}
	}

	// we ignore the UNSPECIFIED specs
	if impact == nil {
		calculator.Add(score)
		return
	}

	// everything else is modify or activate

	if impact.Scoring == explorer.ScoringSystem_IGNORE_SCORE {
		calculator.Add(&Score{
			// We override the type because:
			// 1. If it is set to Result, its value will be added to the total
			// calculation in most calculators despite its weight.
			// 2. We don't want to set it to unscored, because technically we
			// just ignore the score.
			// Thus we set the score to unknown for the sake of the calculator,
			// thus it knows it is handling a scored result, but also knows not
			// to count it.
			Type:            ScoreType_Unknown,
			Value:           score.Value,
			Weight:          0,
			ScoreCompletion: score.ScoreCompletion,
			DataCompletion:  score.DataCompletion,
			DataTotal:       score.DataTotal,
		})
		return
	}

	if impact.Weight > 0 {
		score.Weight = uint32(impact.Weight)
	} else if score.Weight == 0 {
		score.Weight = 1
	}

	calculator.Add(score)
}

func AddDataScore(calculator ScoreCalculator, totalDeps int, finishedDeps int) {
	if totalDeps == 0 {
		return
	}

	dataCompletion := uint32((100 * finishedDeps) / totalDeps)
	calculator.Add(&Score{
		Type:           ScoreType_Unscored,
		DataTotal:      uint32(totalDeps),
		DataCompletion: dataCompletion,
	})
}

func (c *averageScoreCalculator) Add(score *Score) {
	switch score.Type {
	case ScoreType_Skip:
		return
	case ScoreType_Unscored:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal

	case ScoreType_Result:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal
		c.weight += score.Weight

		c.scoreCompletion += score.ScoreCompletion
		c.scoreTotal++

		if score.ScoreCompletion != 0 {
			c.scoreCnt++
			c.value += score.Value
		}
		c.hasResults = true

	case ScoreType_Error:
		c.hasErrors = true

		if c.featureFlagFailErrors {
			// This case is the same as ScoreType_Result. Once the feature flag
			// is removed, this case can be merged with the ScoreType_Result
			c.dataCompletion += score.DataCompletion * score.DataTotal
			c.dataTotal += score.DataTotal
			c.weight += score.Weight

			c.scoreCompletion += score.ScoreCompletion
			c.scoreTotal++

			if score.ScoreCompletion != 0 {
				c.scoreCnt++
				c.value += score.Value
			}
			c.hasResults = true
		} else {
			// This case is the same as ScoreType_Unscored. Once the feature flag
			// is removed, this case can be removed
			c.dataCompletion += score.DataCompletion * score.DataTotal
			c.dataTotal += score.DataTotal
			c.scoreCompletion += score.ScoreCompletion
			c.scoreTotal++
		}

	default:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal
		c.scoreCompletion += score.ScoreCompletion
		c.scoreTotal++
	}
}

func (c *averageScoreCalculator) Calculate() *Score {
	res := &Score{
		Type:      ScoreType_Unscored,
		DataTotal: c.dataTotal,
		// unless we know otherwise, we are setting the data completion to 100
		// until we determine how many datapoints we are looking for
		DataCompletion: 100,
		// if the item is indeed unscored, then the score completion is 100
		// since we are done with the scoring piece
		ScoreCompletion: 100,
	}

	if c.dataTotal != 0 {
		res.DataCompletion = c.dataCompletion / c.dataTotal
	}

	if c.hasResults {
		// if this is scored indicator, we need to calculate the value
		res.Type = ScoreType_Result
		res.ScoreCompletion = c.scoreCompletion / c.scoreTotal
		res.Weight = c.weight
		if c.scoreCnt != 0 {
			res.Value = c.value / c.scoreCnt
		}
	} else if c.hasErrors {
		res.Type = ScoreType_Error
	}

	return res
}

type weightedScoreCalculator struct {
	value                 uint32
	weight                uint32
	scoreTotal            uint32
	scoreCompletion       uint32
	scoreCnt              uint32
	dataTotal             uint32
	dataCompletion        uint32
	hasResults            bool
	hasErrors             bool
	featureFlagFailErrors bool
}

func (c *weightedScoreCalculator) Init() {
	c.value = 0
	c.weight = 0
	c.scoreTotal = 0
	c.scoreCompletion = 0
	c.scoreCnt = 0
	c.dataTotal = 0
	c.dataCompletion = 0
	c.hasResults = false
	c.hasErrors = false
}

func (c *weightedScoreCalculator) Add(score *Score) {
	switch score.Type {
	case ScoreType_Skip:
		return
	case ScoreType_Unscored:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal

	case ScoreType_Result:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal
		c.weight += score.Weight

		c.scoreCompletion += score.ScoreCompletion
		c.scoreTotal++

		if score.ScoreCompletion != 0 {
			c.scoreCnt += score.Weight
			c.value += score.Value * score.Weight
		}
		c.hasResults = true

	case ScoreType_Error:
		c.hasErrors = true
		if c.featureFlagFailErrors {
			// This case is the same as ScoreType_Result. Once the feature flag
			// is removed, this case can be merged with the ScoreType_Result
			c.dataCompletion += score.DataCompletion * score.DataTotal
			c.dataTotal += score.DataTotal
			c.weight += score.Weight

			c.scoreCompletion += score.ScoreCompletion
			c.scoreTotal++

			if score.ScoreCompletion != 0 {
				c.scoreCnt += score.Weight
				c.value += score.Value * score.Weight
			}
			c.hasResults = true
		} else {
			// This case is the same as ScoreType_Unscored. Once the feature flag
			// is removed, this case can be removed
			c.dataCompletion += score.DataCompletion * score.DataTotal
			c.dataTotal += score.DataTotal
			c.scoreCompletion += score.ScoreCompletion
			c.scoreTotal++
		}
	default:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal
		c.scoreCompletion += score.ScoreCompletion
		c.scoreTotal++
	}
}

func (c *weightedScoreCalculator) Calculate() *Score {
	res := &Score{
		Type:      ScoreType_Unscored,
		DataTotal: c.dataTotal,
		// unless we know otherwise, we are setting the data completion to 100
		// until we determine how many datapoints we are looking for
		DataCompletion: 100,
		// if the item is indeed unscored, then the score completion is 100
		// since we are done with the scoring piece
		ScoreCompletion: 100,
	}

	if c.dataTotal != 0 {
		res.DataCompletion = c.dataCompletion / c.dataTotal
	}

	if c.hasResults {
		res.Type = ScoreType_Result
		res.ScoreCompletion = c.scoreCompletion / c.scoreTotal
		res.Weight = c.weight
		res.Value = c.value / c.scoreCnt
	} else if c.hasErrors {
		res.Type = ScoreType_Error
	}

	return res
}

type worstScoreCalculator struct {
	value                 uint32
	weight                uint32
	scoreTotal            uint32
	scoreCompletion       uint32
	dataTotal             uint32
	dataCompletion        uint32
	hasResults            bool
	hasErrors             bool
	featureFlagFailErrors bool
}

func (c *worstScoreCalculator) Init() {
	c.value = 100
	c.weight = 0
	c.scoreTotal = 0
	c.scoreCompletion = 0
	c.dataTotal = 0
	c.dataCompletion = 0
	c.hasResults = false
	c.hasErrors = false
}

func (c *worstScoreCalculator) Add(score *Score) {
	switch score.Type {
	case ScoreType_Skip:
		return
	case ScoreType_Unscored:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal

	case ScoreType_Result:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal
		c.weight += score.Weight

		c.scoreTotal++
		c.scoreCompletion += score.ScoreCompletion

		if score.ScoreCompletion != 0 && score.Weight != 0 && score.Value < c.value {
			c.value = score.Value
		}
		c.hasResults = true

	case ScoreType_Error:
		c.hasErrors = true

		if c.featureFlagFailErrors {
			// This case is the same as ScoreType_Result. Once the feature flag
			// is removed, this case can be merged with the ScoreType_Result
			c.dataCompletion += score.DataCompletion * score.DataTotal
			c.dataTotal += score.DataTotal
			c.weight += score.Weight

			c.scoreTotal++
			c.scoreCompletion += score.ScoreCompletion

			if score.ScoreCompletion != 0 && score.Weight != 0 && score.Value < c.value {
				c.value = score.Value
			}
			c.hasResults = true
		} else {
			// This case is the same as ScoreType_Unscored. Once the feature flag
			// is removed, this case can be removed
			c.dataCompletion += score.DataCompletion * score.DataTotal
			c.dataTotal += score.DataTotal
			c.scoreCompletion += score.ScoreCompletion
			c.scoreTotal++
		}

	default:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal
		c.scoreCompletion += score.ScoreCompletion
		c.scoreTotal++
	}
}

func (c *worstScoreCalculator) Calculate() *Score {
	res := &Score{
		Type:      ScoreType_Unscored,
		DataTotal: c.dataTotal,
		// unless we know otherwise, we are setting the data completion to 100
		// until we determine how many datapoints we are looking for
		DataCompletion: 100,
		// if the item is indeed unscored, then the score completion is 100
		// since we are done with the scoring piece
		ScoreCompletion: 100,
	}
	if c.dataTotal != 0 {
		res.DataCompletion = c.dataCompletion / c.dataTotal
	}

	if c.scoreTotal == 0 {
		return res
	}

	if c.hasResults {
		res.Type = ScoreType_Result
		res.ScoreCompletion = c.scoreCompletion / c.scoreTotal
		res.Weight = c.weight
		res.Value = c.value
	} else if c.hasErrors {
		res.Type = ScoreType_Error
	}

	return res
}

type scoreCalculatorOptions struct {
	featureFlagFailErrors bool
}

// ScoreCalculatorOption is a function that sets some option on a score calculator
type ScoreCalculatorOption func(*scoreCalculatorOptions)

// WithScoreCalculatorFeatureFlagFailErrors sets the feature flag fail errors option
func WithScoreCalculatorFeatureFlagFailErrors() ScoreCalculatorOption {
	return func(o *scoreCalculatorOptions) {
		o.featureFlagFailErrors = true
	}
}

// NewScoreCalculator returns a score calculator based on a scoring system
func NewScoreCalculator(scoringSystem explorer.ScoringSystem, opts ...ScoreCalculatorOption) (ScoreCalculator, error) {
	var res ScoreCalculator

	options := scoreCalculatorOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	switch scoringSystem {
	case explorer.ScoringSystem_AVERAGE, explorer.ScoringSystem_SCORING_UNSPECIFIED, explorer.ScoringSystem_DATA_ONLY:
		res = &averageScoreCalculator{
			featureFlagFailErrors: options.featureFlagFailErrors,
		}
	case explorer.ScoringSystem_WEIGHTED:
		res = &weightedScoreCalculator{
			featureFlagFailErrors: options.featureFlagFailErrors,
		}
	case explorer.ScoringSystem_WORST:
		res = &worstScoreCalculator{
			featureFlagFailErrors: options.featureFlagFailErrors,
		}
	default:
		return nil, errors.New("don't know how to create scoring calculator for system " + strconv.Itoa(int(scoringSystem)))
	}
	res.Init()
	return res, nil
}
