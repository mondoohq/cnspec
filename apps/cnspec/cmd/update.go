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
	updateCmd.Flags().String("channel", "", "Release channel to update from: stable or preview (default: the configured update_channel, or the channel this build belongs to)")
}

// envUpdateReexec marks the process that selfupdate exec'd into after installing
// a new binary. It is set just before the update and inherited across the exec,
// which is what lets the successor tell "I am the result of an update" apart
// from "someone switched engine updates off".
const envUpdateReexec = "MONDOO_CNSPEC_UPDATE_REEXEC"

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Hidden: true,
	Use:    "update",
	Short:  "Update cnspec to the latest version",
	Long: `Update the cnspec binary to the latest release.

This downloads the release for this platform, verifies it, and re-executes the
new binary. It is the same mechanism cnspec uses to update itself in the
background; running this command performs the check immediately instead of
waiting for the next refresh interval.

--channel updates from a different release channel for this one command,
without changing any configuration. On a stable install that is how you move to
a release candidate without putting the machine on the pre-release track
permanently.

It cannot move you backwards: cnspec only ever updates to a higher version, so
--channel stable on a pre-release build reports that there is nothing newer
rather than rolling back. Reinstall to return to the stable line.

Examples:
  cnspec update                     # the configured channel
  cnspec update --channel preview   # one-off, from the pre-release track`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// we need silence usage here, otherwise we get the usage printed in case of error
		// see https://github.com/spf13/cobra/issues/340
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true

		return runUpdate(cmd)
	},
}

// resolveUpdateChannel returns the channel this run should update from: the
// --channel flag if given, otherwise whatever the configuration and the running
// build resolve to.
//
// An unknown value is an error rather than the fallback it gets in config. In a
// config file the value may be sitting somewhere nobody is looking at, so
// following the build is the safer reading; typed on the command line it is
// worth saying so. Same reasoning as `providers update --channel`.
func resolveUpdateChannel(cmd *cobra.Command) (string, error) {
	flag, _ := cmd.Flags().GetString("channel")
	if flag == "" {
		return config.GetUpdateChannel(), nil
	}

	channel := strings.ToLower(strings.TrimSpace(flag))
	switch channel {
	case config.ChannelStable, config.ChannelPreview:
		return channel, nil
	default:
		return "", errors.Errorf("unknown channel %q, expected %s or %s",
			flag, config.ChannelStable, config.ChannelPreview)
	}
}

func runUpdate(cmd *cobra.Command) error {
	currentVersion := cnspec.GetVersion()

	// This process is the one selfupdate exec'd into after installing the new
	// binary, so the update already happened and this version is its result.
	if os.Getenv(envUpdateReexec) != "" {
		log.Info().Msgf("cnspec updated to %s", currentVersion)
		return nil
	}

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

	config.InitViperConfig()

	channel, err := resolveUpdateChannel(cmd)
	if err != nil {
		return err
	}
	releaseURL := cnspec.ReleaseURL(config.GetUpdatesURL(), channel)

	// selfupdate re-executes the new binary with the current arguments, so the
	// successor runs `update` again. It inherits MONDOO_AUTO_UPDATE_ENGINE=false
	// (ExecUpdatedBinary sets that to break update loops), which would otherwise
	// make it report that updates are disabled immediately after a successful
	// update. Mark the environment so the successor recognizes itself instead.
	// syscall.Exec passes os.Environ() through, so the marker survives the
	// hand-off; the defer covers the paths where no exec happens.
	os.Setenv(envUpdateReexec, "1")
	defer os.Unsetenv(envUpdateReexec)

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

	// Asking a pre-release build for stable finds nothing, because cnspec only
	// ever updates to a higher version. Silence there reads like the channel was
	// ignored, when in fact it was honoured and there is no way down.
	if channel == config.ChannelStable && isPrereleaseVersion(currentVersion) {
		log.Info().Msgf("cnspec %s is a pre-release; the stable channel has nothing newer, and cnspec does not downgrade. Reinstall to return to the stable line.", currentVersion)
		return nil
	}

	// Name the channel when it is not the default. On preview "already the
	// latest version" is true of that track only, and a user who set the channel
	// to get a release candidate needs to be able to tell the difference between
	// "there is nothing newer" and "the channel never took effect".
	if channel != config.ChannelStable {
		log.Info().Msgf("cnspec %s is already the latest version on the %s channel", currentVersion, channel)
		return nil
	}

	log.Info().Msgf("cnspec %s is already the latest version", currentVersion)
	return nil
}

// isPrereleaseVersion reports whether a version carries a semver pre-release
// segment. Build metadata is stripped first: it is not a pre-release under
// SemVer 10. Mirrors the check in cli/config, deliberately as a string split
// rather than a parse, so "unstable" and -rolling builds answer cleanly.
func isPrereleaseVersion(version string) bool {
	core, _, _ := strings.Cut(version, "+")
	_, prerelease, found := strings.Cut(strings.TrimPrefix(core, "v"), "-")
	return found && prerelease != ""
}

// autoUpdateDisabledVia reports which environment variable switches off the
// binary update, or "" when none does.
//
// Deliberately only the environment variables, not the `auto_update` config
// setting, the AutoUpdateEngine feature flag or the --auto-update flag that
// shouldTrySelfUpdate in main also consults. Those govern whether cnspec updates
// itself *without being asked*; running `cnspec update` is the asking, so they
// do not apply. The environment variables are the hard off switch, and one of
// them is how a managed install pins a version, so they still hold here.
func autoUpdateDisabledVia() string {
	for _, env := range []string{selfupdate.EnvAutoUpdate, selfupdate.EnvAutoUpdateEngine} {
		if val := os.Getenv(env); val == "false" || val == "0" {
			return env
		}
	}
	return ""
}
