// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/health"
	"go.mondoo.com/cnspec/v10"
	"go.mondoo.com/cnspec/v10/apps/cnspec/cmd"
)

func main() {
	defer health.ReportPanic("cnspec", cnspec.Version, cnspec.Build)
	cmd.Execute()
}
