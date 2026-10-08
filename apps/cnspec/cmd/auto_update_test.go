// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withAutoUpdateViper sets up a fresh viper the way the root command does: no
// key delimiter, MONDOO_ env prefix with "-" mapped to "_", the --auto-update
// flag bound under both spellings, and the given mondoo.yml content. It
// restores the global viper afterwards.
func withAutoUpdateViper(t *testing.T, configYAML string, args ...string) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.SetOptions(viper.KeyDelimiter("\\"))
	viper.SetEnvPrefix("mondoo")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
	viper.SetConfigType("yaml")
	require.NoError(t, viper.ReadConfig(strings.NewReader(configYAML)))

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool("auto-update", true, "")
	require.NoError(t, viper.BindPFlag("auto-update", flags.Lookup("auto-update")))
	require.NoError(t, viper.BindPFlag("auto_update", flags.Lookup("auto-update")))
	require.NoError(t, flags.Parse(args))
}

func TestAutoUpdateEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config string
		env    string
		args   []string
		want   bool
	}{
		{name: "default is on", want: true},
		{name: "auto_update false in config", config: "auto_update: false\n", want: false},
		// The spelling the docs showed. The binary self-update and the serve
		// readers used to ignore it.
		{name: "auto-update false in config", config: "auto-update: false\n", want: false},
		{name: "auto-update true in config", config: "auto-update: true\n", want: true},
		{name: "canonical spelling wins", config: "auto_update: true\nauto-update: false\n", want: true},
		{name: "env false", env: "false", want: false},
		{name: "env 0", env: "0", want: false},
		{name: "env FALSE", env: "FALSE", want: false},
		{name: "env not a boolean counts as off", env: "off", want: false},
		{name: "env false beats config true", config: "auto_update: true\n", env: "false", want: false},
		{name: "env false beats hyphenated config true", config: "auto-update: true\n", env: "false", want: false},
		{name: "env true beats config false", config: "auto_update: false\n", env: "true", want: true},
		{name: "flag false beats env true", env: "true", args: []string{"--auto-update=false"}, want: false},
		{name: "flag false beats config true", config: "auto_update: true\n", args: []string{"--auto-update=false"}, want: false},
		{name: "flag true beats hyphenated config false", config: "auto-update: false\n", args: []string{"--auto-update=true"}, want: true},
		{name: "unchanged flag default does not override config", config: "auto-update: false\n", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv("MONDOO_AUTO_UPDATE", tc.env)
			}
			withAutoUpdateViper(t, tc.config, tc.args...)
			assert.Equal(t, tc.want, AutoUpdateEnabled())
		})
	}
}
