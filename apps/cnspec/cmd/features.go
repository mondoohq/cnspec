// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/cli/config"
)

// featuresCmd represents the features command
var featuresCmd = &cobra.Command{
	Hidden: true,
	Use:    "features",
	Short:  "Display cnspec features",
	Run: func(cmd *cobra.Command, args []string) {
		// prerequisite: features must be initialized via config on the root command
		// otherwise config.Features won't contain anything useful
		fmt.Println("Active features: " + config.Features.String())

		var optIn []string
		for _, b := range mql.AvailableFeatures {
			f := mql.Feature(b)
			if !config.Features.IsActive(f) {
				optIn = append(optIn, f.String())
			}
		}
		if len(optIn) > 0 {
			fmt.Println("Available (opt-in): " + strings.Join(optIn, ", "))
		}
	},
}

// withRateLimitRetry tells the platform this client waits and retries when it
// is refused as busy, so the platform may refuse instead of queueing.
func withRateLimitRetry(f mql.Features) mql.Features {
	if f.IsActive(mql.RateLimitRetry) {
		return f
	}
	return append(f, byte(mql.RateLimitRetry))
}

func init() {
	rootCmd.AddCommand(featuresCmd)
}
