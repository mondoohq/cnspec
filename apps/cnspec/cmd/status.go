// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	cnquery_app "go.mondoo.com/mql/v13/apps/mql/cmd"
)

func init() {
	rootCmd.AddCommand(cnquery_app.StatusCmd)
}
