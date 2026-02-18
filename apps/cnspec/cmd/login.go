// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	mql_app "go.mondoo.com/mql/v13/apps/mql/cmd"
)

func init() {
	rootCmd.AddCommand(mql_app.LoginCmd)
}
