// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/cli/reportview"
	"go.mondoo.com/cnspec/policy"
)

func init() {
	reportCmd.AddCommand(cmpReportCmd)
	reportCmd.AddCommand(viewReportCmd)
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

// viewReportCmd browses a scan report in the terminal.
//
// The artifact it reads is `--output json-full`, which is the only output format
// that carries the bundle, the resolved policies and the raw results, and
// therefore the only one a viewer can show anything but scores from. Handing it
// a reduced json-v1/json-v2 report is a mistake worth naming: those decode
// "successfully" into a collection with no reports, which looks exactly like a
// scan where every asset failed. reporter.LoadCollectionFile refuses them and
// says which format was given and which is needed.
var viewReportCmd = &cobra.Command{
	Use:   "view <file>",
	Short: "Browse a scan report (--output json-full) in the terminal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		collection, err := loadReportForView(args[0])
		if err != nil {
			// The message is the whole point here: a stack trace tells the user
			// nothing they can act on, while "this is a reduced report, re-run
			// with --output json-full" tells them exactly what to do next.
			log.Error().Msg(err.Error())
			os.Exit(1)
		}

		if !isInteractiveTerminal() {
			log.Error().Msg("the report viewer needs a terminal; try `cnspec report cmp` or read the file directly")
			os.Exit(1)
		}

		// The viewer owns the screen and surfaces its own errors in the footer;
		// keep the logger from scribbling over it.
		prev := zerolog.GlobalLevel()
		zerolog.SetGlobalLevel(zerolog.Disabled)
		err = reportview.RunCollection(collection)
		zerolog.SetGlobalLevel(prev)

		if err != nil {
			log.Error().Err(err).Msg("report viewer failed")
			os.Exit(1)
		}
	},
}

// loadReportForView reads the artifact the viewer is pointed at, turning the
// three ways this goes wrong -- no such file, a directory, and a report in a
// format that does not carry enough to show -- into messages a user can act on.
func loadReportForView(path string) (*policy.ReportCollection, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Newf("no such report: %s", path)
		}
		return nil, errors.Wrapf(err, "cannot read %s", path)
	}
	if info.IsDir() {
		return nil, errors.Newf("%s is a directory; pass the report file itself", path)
	}

	collection, err := reporter.LoadCollectionFile(path)
	if err != nil {
		return nil, err
	}
	return collection, nil
}
