// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

type BlastRadiusIndicator string

const (
	BlastRadius_Small  BlastRadiusIndicator = "s"
	BlastRadius_Medium BlastRadiusIndicator = "m"
	BlastRadius_Large  BlastRadiusIndicator = "l"
)

// BlastRadiusConfig for custom blast radius indicators
type BlastRadiusConfig struct {
	// Percentage of infrastructure in total weight that is considered small.
	// eg. 5%. If something affects 3/100 assets, it is 3% and thus small.
	SmallPct float32
	// Percentage of infrastructure in total weight that is considered medium.
	// eg. 20%. If something affects 10/100 assets, it is 10% and thus medium.
	MediumPct float32
	// Minimum number of assets for something to be considered medium.
	// eg. 10. If something affects 2/4 assets, it is 50%, but still small.
	MediumMinCnt float32
	// Minimum number of assets for something to be considered large.
	// eg. 25. If something affects 20/40 assets, it is 50%, but still medium.
	LargeMinCnt float32
}

// DefaultBlastRadiusConfig
var DefaultBlastRadiusConfig = BlastRadiusConfig{
	SmallPct:     0.05,
	MediumPct:    0.20,
	MediumMinCnt: 10,
	LargeMinCnt:  25,
}

// BlastRadius retrieves the blast radius indicator and assets in this category.
// It requires a weight as input
func (b *BlastRadiusConfig) Indicator(totalWeight float32, weight float32) BlastRadiusIndicator {
	rel := weight / totalWeight
	if rel < b.SmallPct || weight < b.MediumMinCnt {
		return BlastRadius_Small
	}
	if rel < b.MediumPct || weight < b.LargeMinCnt {
		return BlastRadius_Medium
	}
	return BlastRadius_Large
}

func (s *ScoreStats) Add(other *ScoreStats) {
	s.Assets += other.Assets
	s.Critical += other.Critical
	s.High += other.High
	s.Medium += other.Medium
	s.Low += other.Low
	s.None += other.None
	s.Pass += other.Pass
	s.Unknown += other.Unknown
	s.Error += other.Error
	s.Disabled += other.Disabled
	s.Snoozed += other.Snoozed
	s.FirstFailureTime += other.FirstFailureTime
	s.OldestScanTime += other.OldestScanTime
	s.NewestScanTime += other.NewestScanTime
	s.Exceptions += other.Exceptions
}
