// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/health"
	"go.mondoo.com/cnspec/v11"
	"go.mondoo.com/cnspec/v11/apps/cnspec/cmd"
)

func main() {
	defer health.ReportPanic("cnspec", cnspec.Version, cnspec.Build)
	cmd.Execute()
}
