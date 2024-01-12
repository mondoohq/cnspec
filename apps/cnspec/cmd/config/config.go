// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10/cli/config"
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
}

type ScanInterval struct {
	Timer int `json:"timer,omitempty" mapstructure:"timer"`
	Splay int `json:"splay,omitempty" mapstructure:"splay"`
}
