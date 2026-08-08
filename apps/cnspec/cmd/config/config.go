// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
	"go.mondoo.com/mql/v13/cli/config"
)

const (
	DefaultScanIntervalTimer = 60
	DefaultScanIntervalSplay = 60
)

func ReadConfig() (*CliConfig, error) {
	// load viper config into a struct
	var opts CliConfig
	err := viper.Unmarshal(&opts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode into config struct")
	}

	// Backward compatibility: some configs use dotted keys instead of a nested
	// mapping, e.g.
	//   scan_interval.timer: 360
	//   scan_interval.splay: 30
	// cnquery sets viper's key delimiter to "\\" (see mql cli/config), so viper
	// does not expand dotted keys into nested maps and the Unmarshal above leaves
	// these nested fields unset. Detect the dotted form and fold it in.
	if viper.IsSet("scan_interval.timer") || viper.IsSet("scan_interval.splay") {
		if opts.ScanInterval == nil {
			opts.ScanInterval = &ScanInterval{}
		}
		if viper.IsSet("scan_interval.timer") {
			opts.ScanInterval.Timer = viper.GetInt("scan_interval.timer")
		}
		if viper.IsSet("scan_interval.splay") {
			opts.ScanInterval.Splay = viper.GetInt("scan_interval.splay")
		}
	}
	if viper.IsSet("auth.method") {
		if opts.Authentication == nil {
			opts.Authentication = &config.CliConfigAuthentication{}
		}
		opts.Authentication.Method = viper.GetString("auth.method")
	}

	return &opts, nil
}

type CliConfig struct {
	// inherit common config
	config.Config `mapstructure:",squash"`

	// Asset Category
	Category               string `json:"category,omitempty" mapstructure:"category"`
	AutoDetectCICDCategory bool   `json:"detect-cicd,omitempty" mapstructure:"detect-cicd"`

	// Configure scan interval
	ScanInterval *ScanInterval `json:"scan_interval,omitempty" mapstructure:"scan_interval"`

	// BinaryAutoUpdate enables self-update of the cnspec binary itself in serve
	// mode. It is off by default: provider plugins auto-update via the separate
	// auto_update key, but replacing the running binary is opt-in per fleet.
	BinaryAutoUpdate bool `json:"binary_auto_update,omitempty" mapstructure:"binary_auto_update"`

	// Update configures download verification and version policy for both
	// provider and binary updates.
	Update *UpdateConfig `json:"update,omitempty" mapstructure:"update"`
}

type ScanInterval struct {
	Timer int `json:"timer,omitempty" mapstructure:"timer"`
	Splay int `json:"splay,omitempty" mapstructure:"splay"`
}

// UpdateConfig captures the `update:` block in mondoo.yml. Zero values mean
// "use the secure defaults": checksum verification on, signature verification
// in auto mode, and the built-in retention count.
type UpdateConfig struct {
	// VerifySignature is the signature-enforcement policy: "auto" (default,
	// verify when a signature is available), "require" (fail closed when no
	// valid signature), or "off".
	VerifySignature string `json:"verify_signature,omitempty" mapstructure:"verify_signature"`

	// Channel selects the release channel to pull from ("stable" default, or
	// "edge"). Reserved for release-channel routing.
	Channel string `json:"channel,omitempty" mapstructure:"channel"`

	// KeepVersions is how many installed versions to retain per provider/binary
	// for rollback. Zero uses the built-in default.
	KeepVersions int `json:"keep_versions,omitempty" mapstructure:"keep_versions"`

	// Providers holds per-provider version policy, keyed by provider name.
	Providers map[string]ProviderVersionPolicy `json:"providers,omitempty" mapstructure:"providers"`
}

// ProviderVersionPolicy pins or floors a single provider's version.
type ProviderVersionPolicy struct {
	// Pin holds the provider at exactly this version.
	Pin string `json:"pin,omitempty" mapstructure:"pin"`
	// Min refuses any published version below this floor.
	Min string `json:"min,omitempty" mapstructure:"min"`
}
