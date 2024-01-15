// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10/cli/config"
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
}
