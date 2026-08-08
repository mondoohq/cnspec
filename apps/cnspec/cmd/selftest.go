// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"
	"go.mondoo.com/mql/v13"
	cli_errors "go.mondoo.com/mql/v13/cli/errors"
	"go.mondoo.com/mql/v13/providers"
)

func init() {
	rootCmd.AddCommand(selftestCmd)
}

// selftestCmd is a fast, hidden health check used by the self-update flow to
// prove that a freshly downloaded binary can actually start and initialize its
// core runtime before it is activated. It intentionally does no network I/O and
// touches no user data: it verifies the binary's own wiring, not the
// environment. A zero exit code means "this binary is safe to activate".
//
// It is the binary-level counterpart of the provider health check that runs
// before a provider version is activated.
var selftestCmd = &cobra.Command{
	Use:    "selftest",
	Short:  "Verify this binary can start and initialize (used by auto-update)",
	Hidden: true,
	// No config file is required to run a selftest.
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true

		if err := runSelftest(); err != nil {
			// Use a specific non-zero exit code so the updater can tell a failed
			// selftest apart from other errors.
			return cli_errors.NewCommandError(errors.Wrap(err, "selftest failed"), 1)
		}
		return nil
	},
}

// runSelftest exercises the critical startup path: feature initialization and
// construction of the provider runtime. If the binary is broken (wrong
// architecture, missing symbols, a panic during init) this fails to complete.
func runSelftest() error {
	// Initializing features touches the compiled-in feature tables.
	_ = mql.SetFeatures(context.Background(), mql.DefaultFeatures)

	// Constructing and closing the default runtime exercises the provider
	// coordinator wiring without connecting to any asset.
	runtime := providers.DefaultRuntime()
	if runtime == nil {
		return errors.New("failed to construct provider runtime")
	}
	runtime.Close()

	return nil
}
