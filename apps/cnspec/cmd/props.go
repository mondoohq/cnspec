// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"go.mondoo.com/mql/cli/config"
	"gopkg.in/yaml.v3"
)

// scanProps returns the property overrides for a scan. Properties set with
// --props win. Otherwise they come from the `props` key of the config.
//
// Viper lowercases every map key it reads from a config file, so a property
// written as `maxLogins` in the config reaches viper as `maxlogins` and never
// matches the property it was meant to override. The keys are restored to the
// case the config file uses.
func scanProps(flagChanged bool) map[string]string {
	props := viper.GetStringMapString("props")
	if flagChanged || len(props) == 0 {
		return props
	}
	raw, err := rawClientConfig()
	if err != nil {
		log.Debug().Err(err).Msg("could not re-read config to restore the case of props")
		return props
	}
	fileProps, err := configProps(raw)
	if err != nil {
		log.Debug().Err(err).Msg("could not parse props from config")
		return props
	}
	return restorePropsCase(props, fileProps)
}

// rawClientConfig returns the client config as it was read, or nil when no
// config was read from a file or from MONDOO_CONFIG_BASE64.
func rawClientConfig() ([]byte, error) {
	if encoded := os.Getenv("MONDOO_CONFIG_BASE64"); encoded != "" {
		return base64.StdEncoding.DecodeString(encoded)
	}
	if !config.LoadedConfig {
		return nil, nil
	}
	path := viper.ConfigFileUsed()
	if path == "" {
		return nil, nil
	}
	return afero.ReadFile(config.AppFs, path)
}

// configProps reads the `props` key of a client config with its keys as
// written.
func configProps(data []byte) (map[string]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var cfg struct {
		Props map[string]any `yaml:"props"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.Props) == 0 {
		return nil, nil
	}
	res := make(map[string]string, len(cfg.Props))
	for k, v := range cfg.Props {
		if v == nil {
			res[k] = ""
			continue
		}
		res[k] = fmt.Sprint(v)
	}
	return res, nil
}

// restorePropsCase rekeys the props viper resolved (with lowercased keys) to
// the keys of the config file. Values stay the ones viper resolved, so its
// precedence is kept. Keys that only differ in case collapse into one key in
// viper; for those the config file's own values are used. Props that are not
// in the config file keep the key viper has.
func restorePropsCase(resolved map[string]string, fromFile map[string]string) map[string]string {
	if len(fromFile) == 0 {
		return resolved
	}
	byLower := map[string][]string{}
	for k := range fromFile {
		lk := strings.ToLower(k)
		byLower[lk] = append(byLower[lk], k)
	}

	res := make(map[string]string, len(resolved))
	for k, v := range resolved {
		keys, ok := byLower[strings.ToLower(k)]
		if !ok {
			res[k] = v
			continue
		}
		if len(keys) == 1 {
			res[keys[0]] = v
			continue
		}
		for _, orig := range keys {
			res[orig] = fromFile[orig]
		}
	}
	return res
}
