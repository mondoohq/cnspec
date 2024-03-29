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
	SmallPct          float32
	MediumPct         float32
	CategoryThreshold float32
}

// DefaultBlastRadiusConfig
var DefaultBlastRadiusConfig = BlastRadiusConfig{
	SmallPct:          0.05,
	MediumPct:         0.20,
	CategoryThreshold: 20,
}

// BlastRadius retrieves the blast radius indicator and assets in this category.
// It requires a weight as input
func (b *BlastRadiusConfig) Indicator(totalWeight float32, weight float32) BlastRadiusIndicator {
	rel := weight / totalWeight
	if rel < b.SmallPct {
		return BlastRadius_Small
	}
	if rel < b.MediumPct {
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
	s.Pass += other.Pass
	s.Unknown += other.Unknown
	s.Error += other.Error
	s.FirstFailureTime += other.FirstFailureTime
	s.OldestScanTime += other.OldestScanTime
	s.NewestScanTime += other.NewestScanTime
}
