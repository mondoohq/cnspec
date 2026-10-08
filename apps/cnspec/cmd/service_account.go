// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"

	"github.com/spf13/viper"
	"go.mondoo.com/mql/cli/config"
	"go.mondoo.com/mql/providers-sdk/v1/upstream"
)

// verifyServiceAccount checks that the configured service account can
// authenticate before any provider starts. It does not contact Mondoo
// Platform; it only loads the credentials the way every provider does when it
// builds its upstream client. source names where the configuration came from
// and is only used in the error message.
func verifyServiceAccount(creds *upstream.ServiceAccountCredentials, source string) error {
	if creds == nil {
		return nil
	}
	if _, err := upstream.NewServiceAccountRangerPlugin(creds); err != nil {
		return unusableServiceAccountError(source, err)
	}
	return nil
}

func unusableServiceAccountError(source string, cause error) error {
	if source == "" {
		source = "the Mondoo configuration"
	}
	return fmt.Errorf("the Mondoo service account in %s can't be used: %w; "+
		"fix the configuration, run `cnspec login` to register again, "+
		"or set MONDOO_CONFIG_PATH to a different configuration file", source, cause)
}

// configSourceDescription names where the loaded configuration came from, for
// error messages. It never includes configuration content.
func configSourceDescription() string {
	if path := viper.ConfigFileUsed(); config.LoadedConfig && path != "" {
		return path
	}
	if config.Source != "" {
		return "the configuration from " + config.Source
	}
	return ""
}
