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
	"go.mondoo.com/cnspec/v13/apps/cnspec/cmd/interactive"
)

// shouldLaunchInteractive decides whether a bare `cnspec` invocation should
// open the interactive launcher instead of printing help. We only do this when:
//   - there are no arguments at all (just the binary name),
//   - both stdin and stdout are real terminals (so the TUI can draw and read
//     keys, and we are not being piped/redirected), and
//   - the user has not opted out via CNSPEC_NO_TUI.
func shouldLaunchInteractive() bool {
	if len(os.Args) != 1 {
		return false
	}
	if os.Getenv("CNSPEC_NO_TUI") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return false
	}
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return false
	}
	return true
}

// runInteractiveLauncher shows the provider/action picker and, once the user
// chooses a command, re-executes this same binary with the assembled
// arguments. Re-execing (rather than calling into cobra directly) means the
// chosen command goes through the exact same startup path as if the user had
// typed it: provider auto-install, flag parsing, discovery, and all.
func runInteractiveLauncher() {
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
