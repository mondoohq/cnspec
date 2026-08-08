// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"go.mondoo.com/mql/v13/cli/selfupdate"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/utils/verify"

	cnspec_config "go.mondoo.com/cnspec/v13/apps/cnspec/cmd/config"
)

// ApplyUpdateVerification applies the download-verification policy and version
// retention from the mondoo.yml `update:` block to both the provider updater
// and the binary self-updater. It is safe to call before any update runs and is
// a no-op when no update config is present.
//
// It must run before the binary self-update in main(), so a fleet configured
// with verify_signature: require rejects an unsigned binary at the startup
// self-update check, not only inside serve.
func ApplyUpdateVerification(uc *cnspec_config.UpdateConfig) {
	if uc == nil {
		return
	}

	if uc.VerifySignature != "" {
		pol := verify.ParseSignaturePolicy(uc.VerifySignature)
		providers.SetProviderSignaturePolicy(pol)
		selfupdate.SetBinarySignaturePolicy(pol)
	}

	if uc.KeepVersions > 0 {
		providers.SetKeepVersions(uc.KeepVersions)
	}
}

// LoadUpdateVerification reads the config and applies the verification policy,
// swallowing config-read errors (self-update must never block startup on a
// config problem). It is the convenience form used by main() before the binary
// self-update runs.
func LoadUpdateVerification() {
	opts, err := cnspec_config.ReadConfig()
	if err != nil {
		return
	}
	ApplyUpdateVerification(opts.Update)
}
