// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	cnquery_app "go.mondoo.com/cnquery/apps/cnquery/cmd"
)

func init() {
	rootCmd.AddCommand(cnquery_app.ProvidersCmd)
}
