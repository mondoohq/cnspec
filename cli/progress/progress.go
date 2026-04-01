// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package progress

import "go.mondoo.com/mql/v13/providers-sdk/v1/inventory"

// Progress is the single-asset progress reporter used by the policy executor.
type Progress interface {
	OnProgress(current int, total int)
	Score(score string)
	Errored()
	NotApplicable()
	Completed()
}

// Noop is a no-op implementation of Progress.
type Noop struct{}

func (n Noop) OnProgress(int, int) {}
func (n Noop) Score(string)        {}
func (n Noop) Errored()            {}
func (n Noop) NotApplicable()      {}
func (n Noop) Completed()          {}

// MultiProgress is the multi-asset progress reporter used by the scanner.
// Tasks can be added dynamically via AddTask at any point after Open.
type MultiProgress interface {
	Open() error
	AddTask(index string, asset *inventory.Asset)
	OnProgress(index string, percent float64)
	Score(index string, score string)
	Errored(index string)
	NotApplicable(index string)
	Completed(index string)
	Close()
}

// NoopMultiProgress is a no-op implementation of MultiProgress.
type NoopMultiProgress struct{}

func (n NoopMultiProgress) Open() error                      { return nil }
func (n NoopMultiProgress) AddTask(string, *inventory.Asset) {}
func (n NoopMultiProgress) OnProgress(string, float64)       {}
func (n NoopMultiProgress) Score(string, string)             {}
func (n NoopMultiProgress) Errored(string)                   {}
func (n NoopMultiProgress) NotApplicable(string)             {}
func (n NoopMultiProgress) Completed(string)                 {}
func (n NoopMultiProgress) Close()                           {}

// MultiProgressAdapter maps single-asset Progress calls to a keyed MultiProgress.
type MultiProgressAdapter struct {
	Multi MultiProgress
	Key   string
}

func (m *MultiProgressAdapter) OnProgress(current int, total int) {
	percent := 0.0
	if total > 0 {
		percent = float64(current) / float64(total)
	}
	m.Multi.OnProgress(m.Key, percent)
}

func (m *MultiProgressAdapter) Score(score string) { m.Multi.Score(m.Key, score) }
func (m *MultiProgressAdapter) Errored()           { m.Multi.Errored(m.Key) }
func (m *MultiProgressAdapter) NotApplicable()     { m.Multi.NotApplicable(m.Key) }
func (m *MultiProgressAdapter) Completed()         { m.Multi.Completed(m.Key) }
