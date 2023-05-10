package config

import (
	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
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
	cnquery_config.CommonCliConfig `mapstructure:",squash"`

	// Asset Category
	Category               string `json:"category,omitempty" mapstructure:"category"`
	AutoDetectCICDCategory bool   `json:"detect-cicd,omitempty" mapstructure:"detect-cicd"`

	// Configure report sharing
	ShareReport *bool `json:"share_report,omitempty" mapstructure:"share_report"`
}
