// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/apps/cnspec/cmd"
	mql "go.mondoo.com/mql"
	"go.mondoo.com/mql/cli/config"
	"go.mondoo.com/mql/cli/selfupdate"
	"go.mondoo.com/mql/metrics"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/health"

	_ "github.com/glebarez/go-sqlite"

	// Link in all vault backends (AWS, GCP, HashiCorp, keyring) so they
	// self-register with the vault registry. The in-memory backend is always
	// available via the SDK.
	_ "go.mondoo.com/mql/vault/register"
)

func main() {
	defer health.ReportPanic("cnspec", cnspec.Version, cnspec.Build)

	// Before anything resolves a channel. With update_channel unset, a
	// pre-release build follows the pre-release track rather than pulling
	// stable providers built against a different schema.
	config.SetRunningVersion(cnspec.GetVersion())

	// Check if running as 'cnquery' and show deprecation warning
	checkDeprecatedBinaryName()

	// Normalize --auto-update flag to handle both "--auto-update false" and "--auto-update=false" formats
	// This must happen before any argument parsing (self-update check or cobra)
	normalizeAutoUpdateFlag()

	// Check for self-update before anything else
	if shouldTrySelfUpdate() {
		releaseURL := cnspec.ReleaseURL(config.GetUpdatesURL(), config.GetUpdateChannel())
		cfg := selfupdate.Config{
			Enabled:         true,
			RefreshInterval: selfupdate.DefaultRefreshInterval,
			ReleaseURL:      releaseURL,
			BinaryName:      "cnspec",
			CurrentVersion:  cnspec.GetVersion(),
		}
		if updated, err := selfupdate.CheckAndUpdate(cfg); err != nil {
			// Log warning but don't block - only show in debug mode
			if os.Getenv("DEBUG") != "" {
				os.Stderr.WriteString("self-update check failed: " + err.Error() + "\n")
			}
		} else if updated {
			// On Windows, the process was replaced by spawning a new one
			// On Unix, ExecUpdatedBinary doesn't return on success
			return
		}
	}

	go metrics.Start()
	cmd.Execute()
}

// checkDeprecatedBinaryName checks if the tool is being run as 'cnquery' (via symlink,
// hardlink, or renamed binary) and prints a deprecation warning to stderr.
func checkDeprecatedBinaryName() {
	binaryName := filepath.Base(os.Args[0])
	// Remove .exe suffix on Windows
	binaryName = strings.TrimSuffix(binaryName, ".exe")

	if binaryName == "cnquery" {
		log.Warn().Msg("'cnquery' has been renamed to 'mql'. Please update your scripts and aliases. For more information, visit: https://github.com/mondoohq/mql")
	}
}

// normalizeAutoUpdateFlag converts space-separated --auto-update flags to the = format.
// This ensures consistent handling across self-update checks, provider updates, and cobra.
// For example: "--auto-update false" becomes "--auto-update=false"
func normalizeAutoUpdateFlag() {
	newArgs := make([]string, 0, len(os.Args))
	skipNext := false

	for i, arg := range os.Args {
		if skipNext {
			skipNext = false
			continue
		}

		// Handle --auto-update VALUE format (space-separated)
		if arg == "--auto-update" && i+1 < len(os.Args) {
			next := os.Args[i+1]
			// Check if next arg looks like a bool value
			if next == "false" || next == "0" || next == "true" || next == "1" {
				// Convert to = format and skip the next arg
				newArgs = append(newArgs, "--auto-update="+next)
				skipNext = true
				continue
			}
		}

		newArgs = append(newArgs, arg)
	}

	os.Args = newArgs
}

// isSelfUpdateExemptCommand reports whether args invoke a command that must not
// trigger the implicit self-update.
//
// The exemptions fall into three groups:
//
//   - Help and version output. These answer a question about the binary that is
//     already running, and an update would replace that binary before it
//     answers -- so the version printed is not the version the user asked
//     about, and a help request pays for a download. Help is matched anywhere
//     in the arguments, not just in first position, because `cnspec scan
//     --help` is as much a help request as `cnspec --help`. This mirrors the
//     CLI preflight in mql, which likewise treats -h as help and nothing else.
//
//   - `update`. It runs the same self-update itself, and without the refresh
//     interval that paces this one. Letting both run means the implicit check
//     can consume the update before the command sees it, so the command the
//     user actually invoked reports "already the latest version" for work it
//     did not do.
//
//   - `login` and `logout`. login is what configures api_endpoint and
//     updates_url, so an update running ahead of it resolves the release
//     against whatever the machine was pointed at before -- for an install that
//     is being enrolled against a mirror, that is a reach for Mondoo's bucket
//     during the one command that was supposed to stop it. It also re-execs the
//     new binary mid-enrollment, and logout is its counterpart: a machine being
//     unenrolled has no reason to download anything. `cnspec update` remains
//     the explicit way to update, so nothing here removes the ability, only the
//     surprise.
func isSelfUpdateExemptCommand(args []string) bool {
	if len(args) < 2 {
		return false
	}

	for _, arg := range args[1:] {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}

	switch args[1] {
	case "version", "--version", "help",
		"update",
		"login", "register", "logout", "unregister":
		return true
	}

	return false
}

// shouldTrySelfUpdate checks if a self-update should be attempted.
// This uses viper config and CLI flags to determine if auto-update is enabled.
// Note: normalizeAutoUpdateFlag() must be called before this function to convert
// space-separated flags (--auto-update false) to the = format (--auto-update=false).
func shouldTrySelfUpdate() bool {
	// Skip if disabled via environment variable
	// This also prevents infinite loops after an update (the updated process
	// is spawned with MONDOO_AUTO_UPDATE=false)
	if val := os.Getenv(selfupdate.EnvAutoUpdate); val == "false" || val == "0" {
		return false
	}

	if isSelfUpdateExemptCommand(os.Args) {
		return false
	}

	// Initialize viper to read config files (same as detectConnectorName in cli/providers)
	config.InitViperConfig()

	// Check if the AutoUpdateEngine feature flag is enabled
	if !config.GetFeatures().IsActive(mql.AutoUpdateEngine) {
		return false
	}

	// Get auto_update setting from config (defaults to true if not set)
	autoUpdate := config.GetAutoUpdate()

	// Check for --auto-update=VALUE flag (already normalized from space-separated format)
	for _, arg := range os.Args {
		if arg == "--auto-update=false" || arg == "--auto-update=0" {
			return false
		}
		if arg == "--auto-update=true" || arg == "--auto-update=1" {
			return true
		}
	}

	return autoUpdate
}
