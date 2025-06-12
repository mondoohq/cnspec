// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/v11/internal/bundle"
)

func init() {
	rootCmd.AddCommand(bundleCmd)
	bundleCmd.AddCommand(bundleLintCmd)
	bundleCmd.AddCommand(bundlePublishCmd)
	bundleCmd.AddCommand(bundleInitCmd)
	bundleCmd.AddCommand(bundleFormatCmd)
}

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Manage Mondoo Policy Bundles",
}

var bundleLintCmd = &cobra.Command{
	Use:   "lint [path]",
	Short: "Lint a policy bundle",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Implementation already exists in the codebase
		// This is just a placeholder to show the command structure
	},
}

var bundlePublishCmd = &cobra.Command{
	Use:   "publish [path]",
	Short: "Publish a policy bundle",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Implementation already exists in the codebase
	},
}

var bundleInitCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a new policy bundle",
	Run: func(cmd *cobra.Command, args []string) {
		// Implementation already exists in the codebase
	},
}

var bundleFormatCmd = &cobra.Command{
	Use:   "fmt [path]",
	Short: "Format a policy bundle",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sort, _ := cmd.Flags().GetBool("sort")
		queryFmt, _ := cmd.Flags().GetBool("query-fmt")

		path := args[0]

		var err error
		if queryFmt {
			err = bundle.FormatRecursiveWithQueryTitleArray(path, sort)
		} else {
			err = bundle.FormatRecursive(path, sort)
		}

		if err != nil {
			log.Fatal().Err(err).Str("path", path).Msg("failed to format bundle")
		}
	},
}

func init() {
	bundleFormatCmd.Flags().Bool("sort", false, "Sort the bundle contents")
	bundleFormatCmd.Flags().Bool("query-fmt", false, "Format query titles as arrays")
}
