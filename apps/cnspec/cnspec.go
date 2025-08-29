// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"go.mondoo.com/cnquery/v12/metrics"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream/health"
	"go.mondoo.com/cnspec/v12"
	"go.mondoo.com/cnspec/v12/apps/cnspec/cmd"
)

func main() {
	defer health.ReportPanic("cnspec", cnspec.Version, cnspec.Build)
	go metrics.Start()
	cmd.Execute()
}
