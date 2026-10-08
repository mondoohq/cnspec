// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"errors"
	"fmt"
	"sort"

	"github.com/spf13/viper"
	"go.mondoo.com/cnspec/policy"
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

// errBomGenerationFailed is returned when a bill of materials could not be
// generated for at least one asset.
var errBomGenerationFailed = errors.New("bill of materials generation failed")

// bomFailure describes an asset whose bill of materials could not be
// generated.
type bomFailure struct {
	Asset  string
	Reason string
}

// bomFailuresError turns the per-asset failures of a bill of materials run,
// plus the scan errors recorded for assets that never produced a bill of
// materials, into one error. It returns nil when every asset succeeded.
// kind is "SBOM" or "AIBOM".
func bomFailuresError(kind string, failures []bomFailure, scanErrors map[string]string, assetNames map[string]string, generated int) error {
	if len(failures) == 0 && (generated > 0 || len(scanErrors) == 0) {
		return nil
	}

	msg := fmt.Sprintf("could not generate the %s", kind)
	for _, f := range failures {
		msg += fmt.Sprintf("\n  - asset %q: %s", f.Asset, f.Reason)
	}
	mrns := make([]string, 0, len(scanErrors))
	for mrn := range scanErrors {
		mrns = append(mrns, mrn)
	}
	sort.Strings(mrns)
	for _, mrn := range mrns {
		scanErr := scanErrors[mrn]
		name := assetNames[mrn]
		if name == "" {
			name = mrn
		}
		msg += fmt.Sprintf("\n  - asset %q: scan error: %s", name, scanErr)
	}
	return fmt.Errorf("%w: %s", errBomGenerationFailed, msg)
}

// assetNamesByMrn maps the asset MRNs of a report collection to asset names,
// so scan errors (keyed by MRN) can name the asset.
func assetNamesByMrn(report *policy.ReportCollection) map[string]string {
	names := map[string]string{}
	for mrn, asset := range report.GetAssets() {
		names[mrn] = asset.GetName()
	}
	return names
}
