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

// withInstallPolicyRequirementsViper sets up a fresh viper the way the root
// command does (MONDOO_ env prefix with "-" mapped to "_"), binds the scan
// flag the way scan's PreRun does, and loads the given mondoo.yml content. It
// restores the global viper afterwards.
func withInstallPolicyRequirementsViper(t *testing.T, configYAML string, args ...string) {
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
	flags.Bool(installPolicyRequirementsFlag, false, "")
	require.NoError(t, viper.BindPFlag(installPolicyRequirementsFlag, flags.Lookup(installPolicyRequirementsFlag)))
	require.NoError(t, flags.Parse(args))
}

func TestInstallPolicyRequirementsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config string
		env    string
		args   []string
		want   bool
	}{
		{name: "default is off", want: false},
		{name: "flag on", args: []string{"--install-policy-requirements"}, want: true},
		{name: "flag false", args: []string{"--install-policy-requirements=false"}, want: false},
		{name: "env true", env: "true", want: true},
		{name: "env 1", env: "1", want: true},
		{name: "env false", env: "false", want: false},
		{name: "env not a boolean counts as off", env: "yes", want: false},
		{name: "config true", config: "install-policy-requirements: true\n", want: true},
		{name: "env false beats config true", config: "install-policy-requirements: true\n", env: "false", want: false},
		{name: "flag false beats env true", env: "true", args: []string{"--install-policy-requirements=false"}, want: false},
		// serve has no such flag and scan leaves it unset: its default must not
		// hide the environment variable.
		{name: "unchanged flag default does not override env", env: "true", want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv("MONDOO_INSTALL_POLICY_REQUIREMENTS", tc.env)
			}
			withInstallPolicyRequirementsViper(t, tc.config, tc.args...)
			assert.Equal(t, tc.want, installPolicyRequirementsEnabled())
		})
	}
}
