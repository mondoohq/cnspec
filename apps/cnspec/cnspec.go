// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"go.mondoo.com/mql/v13/metrics"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/health"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/cnspec/v13/apps/cnspec/cmd"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	defer health.ReportPanic("cnspec", cnspec.Version, cnspec.Build)
	go metrics.Start()
	cmd.Execute()
}
