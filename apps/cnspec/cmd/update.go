// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/mql/cli/config"
	"go.mondoo.com/mql/cli/selfupdate"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

// defaultReleaseURL is where cnspec looks for the latest release manifest. It
// matches the URL the implicit auto-update in main uses, so an explicit update
// and a background one resolve the same release.
const defaultReleaseURL = "https://releases.mondoo.com/cnspec/latest.json"

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Hidden: true,
	Use:    "update",
	Short:  "Update cnspec to the latest version",
	Long: `Update the cnspec binary to the latest release.

This downloads the release for this platform, verifies it, and re-executes the
new binary. It is the same mechanism cnspec uses to update itself in the
background; running this command performs the check immediately instead of
waiting for the next refresh interval.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// we need silence usage here, otherwise we get the usage printed in case of error
		// see https://github.com/spf13/cobra/issues/340
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true

		return runUpdate()
	},
}

func runUpdate() error {
	currentVersion := cnspec.GetVersion()

	// selfupdate skips these cases and returns (false, nil), which would be
	// indistinguishable from "already up to date". The user asked for an update
	// explicitly, so say why nothing is going to happen instead.
	// "unstable" is what a build without version ldflags reports, and selfupdate
	// only screens for the "-rolling" suffix - an unstable build reaches the
	// version comparison and fails there with a semver parse error instead.
	if currentVersion == "unstable" || strings.HasSuffix(currentVersion, "-rolling") {
		return errors.Errorf("cannot update a development build (version %s), install a release build instead", currentVersion)
	}
	if disabledVia := autoUpdateDisabledVia(); disabledVia != "" {
		return errors.Errorf("updates are disabled via %s, unset it to update", disabledVia)
	}

	releaseURL := defaultReleaseURL
	config.InitViperConfig()
	if updatesURL := config.GetUpdatesURL(); updatesURL != "" {
		releaseURL = updatesURL + "/cnspec/latest.json"
	}

	// selfupdate re-executes the new binary with the current os.Args. Left alone
	// that would re-run `update` in the new process, which then finds engine
	// updates switched off (ExecUpdatedBinary sets that to break update loops)
	// and would report the wrong reason for doing nothing. Point the successor
	// at `version` instead, so a completed update ends by printing what is now
	// installed. On Unix the exec never returns, so restoring os.Args only
	// matters on the paths that do.
	origArgs := os.Args
	os.Args = []string{origArgs[0], "version"}
	defer func() { os.Args = origArgs }()

	updated, err := selfupdate.CheckAndUpdate(selfupdate.Config{
		Enabled: true,
		// An explicit update must not be throttled by the refresh interval that
		// paces the background check, otherwise running this command inside that
		// window silently does nothing.
		RefreshInterval: 0,
		ReleaseURL:      releaseURL,
		BinaryName:      "cnspec",
		CurrentVersion:  currentVersion,
	})
	if err != nil {
		return errors.Wrap(err, "failed to update cnspec")
	}

	// On Unix the process has already been replaced by the new binary, so
	// reaching here with updated==true means the Windows path spawned it.
	if updated {
		log.Info().Msg("cnspec was updated")
		return nil
	}

	log.Info().Msgf("cnspec %s is already the latest version", currentVersion)
	return nil
}

// autoUpdateDisabledVia reports which environment variable switches off the
// binary update, or "" when none does.
func autoUpdateDisabledVia() string {
	for _, env := range []string{selfupdate.EnvAutoUpdate, selfupdate.EnvAutoUpdateEngine} {
		if val := os.Getenv(env); val == "false" || val == "0" {
			return env
		}
	}
	return ""
}
