// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"math"
	"strconv"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v11/explorer"
	"google.golang.org/protobuf/proto"
)

// ScoreCalculator interface for calculating scores
type ScoreCalculator interface {
	Add(score *Score, impact *explorer.Impact)
	Calculate() *Score
	Init()
	String() string
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

func (c *averageScoreCalculator) String() string {
	return "Average"
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
		}, nil)
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
		calculator.Add(score, nil)
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
		}, nil)
		return
	}

	if impact.Weight > 0 {
		score.Weight = uint32(impact.Weight)
	} else if score.Weight == 0 {
		score.Weight = 1
	}

	calculator.Add(score, impact)
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
	}, nil)
}

func (c *averageScoreCalculator) Add(score *Score, impact *explorer.Impact) {
	switch score.Type {
	case ScoreType_Skip, ScoreType_Disabled, ScoreType_OutOfScope:
		return
	case ScoreType_Unscored:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal

	case ScoreType_Result:
		if impact != nil && (impact.Action == explorer.Action_IGNORE || impact.Action == explorer.Action_DEACTIVATE) {
			return
		}

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

func (c *weightedScoreCalculator) String() string {
	return "Weighted Average"
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

func (c *weightedScoreCalculator) Add(score *Score, impact *explorer.Impact) {
	switch score.Type {
	case ScoreType_Skip, ScoreType_Disabled, ScoreType_OutOfScope:
		return
	case ScoreType_Unscored:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal

	case ScoreType_Result:
		if impact != nil && (impact.Action == explorer.Action_IGNORE || impact.Action == explorer.Action_DEACTIVATE) {
			return
		}

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

func (c *worstScoreCalculator) String() string {
	return "Highest Impact"
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

func (c *worstScoreCalculator) Add(score *Score, impact *explorer.Impact) {
	switch score.Type {
	case ScoreType_Skip, ScoreType_Disabled, ScoreType_OutOfScope:
		return
	case ScoreType_Unscored:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal

	case ScoreType_Result:
		if impact != nil && (impact.Action == explorer.Action_IGNORE || impact.Action == explorer.Action_DEACTIVATE) {
			return
		}

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

type bandedScoreCalculator struct {
	crit    uint32
	high    uint32
	mid     uint32
	low     uint32
	critMax uint32
	highMax uint32
	midMax  uint32
	lowMax  uint32

	minscore              uint32
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

func (c *bandedScoreCalculator) String() string {
	return "Banded"
}

func (c *bandedScoreCalculator) Init() {
	c.minscore = 100
	c.value = 100
	c.weight = 0
	c.scoreTotal = 0
	c.scoreCompletion = 0
	c.dataTotal = 0
	c.dataCompletion = 0
	c.hasResults = false
	c.hasErrors = false
}

func (c *bandedScoreCalculator) Add(score *Score, impact *explorer.Impact) {
	switch score.Type {
	case ScoreType_Skip, ScoreType_OutOfScope, ScoreType_Disabled:
		return
	case ScoreType_Unscored:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal

	case ScoreType_Result:
		if impact != nil && (impact.Action == explorer.Action_IGNORE || impact.Action == explorer.Action_DEACTIVATE) {
			return
		}

		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal
		c.weight += score.Weight

		c.scoreTotal++
		c.scoreCompletion += score.ScoreCompletion

		if score.ScoreCompletion != 0 && score.Weight != 0 {
			category := uint32(0)
			if impact != nil {
				// Store pointer to avoid panic
				if impactV := impact.GetValue(); impactV != nil {
					if value := impactV.GetValue(); value < 100 && value > 0 {
						category = 100 - uint32(value)
					}
				}
			}

			if category <= 10 {
				c.critMax += score.Weight
				if score.Value < 100 {
					c.crit += score.Weight
				}
			} else if category <= 30 {
				c.highMax += score.Weight
				if score.Value < 100 {
					c.high += score.Weight
				}
			} else if category <= 60 {
				c.midMax += score.Weight
				if score.Value < 100 {
					c.mid += score.Weight
				}
			} else {
				c.lowMax += score.Weight
				if score.Value < 100 {
					c.low += score.Weight
				}
			}
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

func (c *bandedScoreCalculator) Calculate() *Score {
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

		pcrLow := float64(1)
		if c.lowMax != 0 {
			pcrLow = float64(c.lowMax-c.low) / float64(c.lowMax)
		}
		fMid := float64(1)
		if c.midMax != 0 {
			fMid = float64(c.midMax-c.mid) / float64(c.midMax)
		}
		pcrMid := (3 + pcrLow) / 4 * fMid
		fHigh := float64(1)
		if c.highMax != 0 {
			fHigh = float64(c.highMax-c.high) / float64(c.highMax)
		}
		pcrHigh := (1 + pcrMid) / 2 * fHigh
		fCrit := float64(1)
		if c.critMax != 0 {
			fCrit = float64(c.critMax-c.crit) / float64(c.critMax)
		}
		pcrCrit := (1 + 4*pcrHigh) / 5 * fCrit

		if c.crit != 0 {
			res.Value = uint32(math.Floor(float64(50) * pcrCrit))
		} else if c.high != 0 {
			res.Value = uint32(math.Floor(float64(50)*pcrHigh)) + 10
		} else if c.mid != 0 {
			res.Value = uint32(math.Floor(float64(50)*pcrMid)) + 30
		} else if c.low != 0 {
			res.Value = uint32(math.Floor(float64(40)*pcrLow)) + 60
		} else {
			res.Value = 100
		}
		res.ScoreCompletion = c.scoreCompletion / c.scoreTotal
		res.Weight = c.weight
	} else if c.hasErrors {
		res.Type = ScoreType_Error
	}

	return res
}

type decayedScoreCalculator struct {
	x                     float64
	xmax                  float64
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

func (c *decayedScoreCalculator) String() string {
	return "Decayed"
}

var gravity float64 = 10

func (c *decayedScoreCalculator) Init() {
	c.x = 0
	c.xmax = 0
	c.value = 100
	c.weight = 0
	c.scoreTotal = 0
	c.scoreCompletion = 0
	c.dataTotal = 0
	c.dataCompletion = 0
	c.hasResults = false
	c.hasErrors = false
}

func (c *decayedScoreCalculator) Add(score *Score, impact *explorer.Impact) {
	switch score.Type {
	case ScoreType_Skip, ScoreType_OutOfScope, ScoreType_Disabled:
		return
	case ScoreType_Unscored:
		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal

	case ScoreType_Result:
		if impact != nil && (impact.Action == explorer.Action_IGNORE || impact.Action == explorer.Action_DEACTIVATE) {
			return
		}

		c.dataCompletion += score.DataCompletion * score.DataTotal
		c.dataTotal += score.DataTotal
		c.weight += score.Weight

		c.scoreTotal++
		c.scoreCompletion += score.ScoreCompletion

		if score.ScoreCompletion != 0 && score.Weight != 0 {
			// TODO: we can add an optional accelerator here later on.
			// The function changes to v := math.Pow( ... , accelerator)
			// with accelerator > 0, default = 1
			v := float64(100-score.Value) / 100
			c.x += v * float64(score.Weight)
			if impact.Value == nil {
				c.xmax += 1 * float64(score.Weight)
			} else {
				c.xmax += float64(impact.Value.Value) / 100 * float64(score.Weight)
			}
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

func (c *decayedScoreCalculator) Calculate() *Score {
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
		relGravity := float64(c.weight) / gravity
		xscaled := c.x / c.xmax * (relGravity)
		floor := math.Exp(-relGravity)
		res.Value = uint32(math.Floor(100 * (math.Exp(-xscaled) - floor) / (1 - floor)))
		res.ScoreCompletion = c.scoreCompletion / c.scoreTotal
		res.Weight = c.weight
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
	case explorer.ScoringSystem_BANDED:
		res = &bandedScoreCalculator{
			featureFlagFailErrors: options.featureFlagFailErrors,
		}
	case explorer.ScoringSystem_DECAYED:
		res = &decayedScoreCalculator{
			featureFlagFailErrors: options.featureFlagFailErrors,
		}
	default:
		return nil, errors.New("don't know how to create scoring calculator for system " + strconv.Itoa(int(scoringSystem)))
	}
	res.Init()
	return res, nil
}
