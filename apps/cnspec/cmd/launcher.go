// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/v13/apps/cnspec/cmd/interactive"
)

func init() {
	rootCmd.AddCommand(uiCmd)
}

// uiCmd is the interactive launcher: a searchable, categorized picker over
// every provider/connector cnspec supports, plus the action to run against it.
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

// runInteractiveLauncher shows the provider/action picker and, once the user
// chooses a command, re-executes this same binary with the assembled
// arguments. Re-execing (rather than calling into cobra directly) means the
// chosen command goes through the exact same startup path as if the user had
// typed it: provider auto-install, flag parsing, discovery, and all.
func runInteractiveLauncher() {
	if !isInteractiveTerminal() {
		log.Error().Msg("the interactive launcher needs a terminal; try `cnspec scan local` or `cnspec --help`")
		os.Exit(1)
	}

	// The launcher owns the screen; keep the logger from scribbling on the
	// alt-screen while the catalog loads.
	prev := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	res, err := interactive.Run()
	zerolog.SetGlobalLevel(prev)
	if err != nil {
		log.Error().Err(err).Msg("interactive launcher failed")
		os.Exit(1)
	}
	if !res.Launched || len(res.Args) == 0 {
		return
	}

	self, err := os.Executable()
	if err != nil {
		log.Error().Err(err).Msg("could not locate the cnspec binary to launch the command")
		os.Exit(1)
	}

	// Show the user exactly what is about to run before handing off.
	fmt.Fprintf(os.Stdout, "\n$ cnspec %s\n\n", strings.Join(res.Args, " "))

	c := exec.Command(self, res.Args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = os.Environ()
	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Error().Err(err).Msg("failed to run the selected command")
		os.Exit(1)
	}
}
