// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"os"

	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/cnspec/v13/apps/cnspec/cmd"
	mql "go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/cli/config"
	"go.mondoo.com/mql/v13/cli/selfupdate"
	"go.mondoo.com/mql/v13/metrics"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/health"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	defer health.ReportPanic("cnspec", cnspec.Version, cnspec.Build)

	// Normalize --auto-update flag to handle both "--auto-update false" and "--auto-update=false" formats
	// This must happen before any argument parsing (self-update check or cobra)
	normalizeAutoUpdateFlag()

	// Check for self-update before anything else
	if shouldTrySelfUpdate() {
		releaseURL := "https://releases.mondoo.com/cnspec/latest.json"
		if updatesURL := config.GetUpdatesURL(); updatesURL != "" {
			releaseURL = updatesURL + "/cnspec/latest.json"
		}
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

	// Skip for version/help commands - these should run fast
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "help", "--help", "-h", "--version":
			return false
		}
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
