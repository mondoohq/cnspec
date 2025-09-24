// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/v12"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the cnspec version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cnspec.Info())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
