// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/apps/cnspec/cmd/interactive"
)

func init() {
	rootCmd.AddCommand(uiCmd)
}

// uiCmd is the interactive launcher: a searchable, categorized list of every
// provider/connector cnspec supports next to a detail pane that shows what can
// be done with the selected one. Commands run from inside the launcher, so a
// session can run several.
//
// It is intentionally Hidden for now so we can ship and dogfood it as an
// explicit `cnspec ui` without changing what a bare `cnspec` does. Once it is
// stable, the plan is to make a bare `cnspec` in a terminal open this launcher.
var uiCmd = &cobra.Command{
	Use:     "ui",
	Aliases: []string{"menu", "interactive", "launch"},
	Short:   "Interactive launcher to discover providers and actions (experimental)",
	Hidden:  true,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runInteractiveLauncher()
	},
}

// isInteractiveTerminal reports whether both stdin and stdout are real
// terminals, so the TUI can draw and read keys.
func isInteractiveTerminal() bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	stdoutTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	stdinTTY := isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
	return stdoutTTY && stdinTTY
}

func runInteractiveLauncher() {
	if !isInteractiveTerminal() {
		log.Error().Msg("the interactive launcher needs a terminal; try `cnspec scan local` or `cnspec --help`")
		os.Exit(1)
	}

	// The launcher owns the screen and surfaces its own errors in the footer;
	// keep the logger from scribbling over it. Commands launched from the
	// launcher run in a child process with its own logger, so this does not
	// quiet them.
	prev := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	err := interactive.Run()
	zerolog.SetGlobalLevel(prev)

	if err != nil {
		log.Error().Err(err).Msg("interactive launcher failed")
		os.Exit(1)
	}
}
