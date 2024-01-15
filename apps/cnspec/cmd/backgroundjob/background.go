// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backgroundjob

import (
	"time"
)

const (
	// Service Name
	SvcName = "cnspec" // NOTE: this name needs to align with the service name in packages
)

type JobRunner func() error

func New(timer, splay time.Duration) (*BackgroundScanner, error) {
	return &BackgroundScanner{
		timer: timer,
		splay: splay,
	}, nil
}

type BackgroundScanner struct {
	timer time.Duration
	splay time.Duration
}

func (bs *BackgroundScanner) Run(runScanFn JobRunner) error {
	Serve(
		bs.timer,
		bs.splay,
		runScanFn)
	return nil
}
