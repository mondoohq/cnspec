// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/v12/cli/reporter"
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
	Use:   "cmp <expected> <actual>",
	Short: "Compare cnspec reports",
	Run: func(cmd *cobra.Command, args []string) {
		expected := args[0]
		actual := args[1]

		expectedReport, err := reporter.FromSingleFile(expected)
		if err != nil {
			log.Fatal().Err(err).Str("expected", expected).Msg("failed to load expected report")
		}

		compareReport, err := reporter.FromSingleFile(actual)
		if err != nil {
			log.Fatal().Err(err).Str("actual", actual).Msg("failed to load actual report")
		}

		equal := reporter.CompareReports(expectedReport, compareReport)
		// return 1 if the reports are not equal
		if !equal {
			os.Exit(1)
		}
	},
}
