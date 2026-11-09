// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/base64"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/cli/config"
)

const camelCasePropsConfig = `
props:
  maxLogins: "10"
  allowedShells: /bin/bash
  lower: 1
`

// loadTestConfig loads data as the client config, the way cnspec does at
// startup, and restores the global state when the test ends.
func loadTestConfig(t *testing.T, data string) {
	t.Helper()
	prevFs, prevLoaded := config.AppFs, config.LoadedConfig
	t.Cleanup(func() {
		config.AppFs, config.LoadedConfig = prevFs, prevLoaded
		viper.Reset()
	})

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "/etc/mondoo/mondoo.yml", []byte(data), 0o600))
	config.AppFs = fs

	viper.Reset()
	viper.SetFs(fs)
	viper.SetConfigType("yaml")
	viper.SetOptions(viper.KeyDelimiter("\\"))
	viper.SetConfigFile("/etc/mondoo/mondoo.yml")
	require.NoError(t, viper.ReadInConfig())
	config.LoadedConfig = true
}

func TestScanProps_ConfigFileKeepsCase(t *testing.T) {
	loadTestConfig(t, camelCasePropsConfig)

	// this is what viper hands out on its own
	assert.Equal(t, map[string]string{
		"maxlogins":     "10",
		"allowedshells": "/bin/bash",
		"lower":         "1",
	}, viper.GetStringMapString("props"))

	assert.Equal(t, map[string]string{
		"maxLogins":     "10",
		"allowedShells": "/bin/bash",
		"lower":         "1",
	}, scanProps(false))
}

func TestScanProps_FlagWins(t *testing.T) {
	loadTestConfig(t, camelCasePropsConfig)
	viper.Set("props", map[string]string{"maxLogins": "5"})

	assert.Equal(t, map[string]string{"maxLogins": "5"}, scanProps(true))
}

func TestScanProps_Base64Config(t *testing.T) {
	loadTestConfig(t, camelCasePropsConfig)
	config.LoadedConfig = false
	t.Setenv("MONDOO_CONFIG_BASE64", base64.StdEncoding.EncodeToString([]byte(camelCasePropsConfig)))

	assert.Equal(t, "10", scanProps(false)["maxLogins"])
}

func TestScanProps_NoProps(t *testing.T) {
	loadTestConfig(t, "log:\n  level: info\n")
	assert.Empty(t, scanProps(false))
}

func TestConfigProps(t *testing.T) {
	props, err := configProps([]byte(`
props:
  maxLogins: 10
  enabled: true
  empty:
  name: value
other: 1
`))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"maxLogins": "10",
		"enabled":   "true",
		"empty":     "",
		"name":      "value",
	}, props)

	props, err = configProps(nil)
	require.NoError(t, err)
	assert.Nil(t, props)

	_, err = configProps([]byte("props: [\n"))
	assert.Error(t, err)
}

func TestRestorePropsCase(t *testing.T) {
	t.Run("rekeys to the file's case and keeps resolved values", func(t *testing.T) {
		res := restorePropsCase(
			map[string]string{"maxlogins": "from-env", "other": "x"},
			map[string]string{"maxLogins": "10"},
		)
		assert.Equal(t, map[string]string{"maxLogins": "from-env", "other": "x"}, res)
	})

	t.Run("keys that only differ in case use the file's values", func(t *testing.T) {
		res := restorePropsCase(
			map[string]string{"maxlogins": "2"},
			map[string]string{"maxLogins": "1", "MaxLogins": "2"},
		)
		assert.Equal(t, map[string]string{"maxLogins": "1", "MaxLogins": "2"}, res)
	})

	t.Run("nothing in the file", func(t *testing.T) {
		in := map[string]string{"a": "b"}
		assert.Equal(t, in, restorePropsCase(in, nil))
	})
}
