// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/v11/cli/reporter"
)

func init() {
	reportCmd.AddCommand(cmpReportCmd)
	rootCmd.AddCommand(reportCmd)
}

var reportCmd = &cobra.Command{
	Use:    "report",
	Short:  "Report commands (Experimental)",
	Hidden: true,
}

var cmpReportCmd = &cobra.Command{
	Use:   "cmp <expected> <compare>",
	Short: "Compare cnspec reports",
	Run: func(cmd *cobra.Command, args []string) {
		base := args[0]
		compare := args[1]

		expectedReport, err := reporter.FromSingleFile(base)
		if err != nil {
			log.Fatal().Err(err).Str("base", base).Msg("failed to load base report")
		}

		compareReport, err := reporter.FromSingleFile(compare)
		if err != nil {
			log.Fatal().Err(err).Str("base", base).Msg("failed to load base report")
		}

		equal := reporter.CompareReports(expectedReport, compareReport)
		// return 1 if the reports are not equal
		if !equal {
			os.Exit(1)
		}
	},
}
