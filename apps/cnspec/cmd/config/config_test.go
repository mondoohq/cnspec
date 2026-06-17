// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/spf13/viper"
)

func TestConfigParsing(t *testing.T) {
	data := `
agent_mrn: //agents.api.mondoo.app/spaces/musing-saha-952142/agents/1zDY7auR20SgrFfiGUT5qZWx6mE
api_endpoint: https://us.api.mondoo.com
certificate: |
  -----BEGIN CERTIFICATE-----
  MIICV .. fis=
  -----END CERTIFICATE-----

mrn: //agents.api.mondoo.app/spaces/musing-saha-952142/serviceaccounts/1zDY7cJ7bA84JxxNBWDxBdui2xE
private_key: |
  -----BEGIN PRIVATE KEY-----
  MIG2AgE....C0Dvs=
  -----END PRIVATE KEY-----
space_mrn: //captain.api.mondoo.app/spaces/musing-saha-952142

scan_interval:
  timer: 10
  splay: 20
`

	viper.SetConfigType("yaml")
	_ = viper.ReadConfig(strings.NewReader(data))

	cfg, err := ReadConfig()
	require.NoError(t, err)
	assert.Equal(t, "//agents.api.mondoo.app/spaces/musing-saha-952142/agents/1zDY7auR20SgrFfiGUT5qZWx6mE", cfg.AgentMrn)
	assert.Equal(t, "//agents.api.mondoo.app/spaces/musing-saha-952142/serviceaccounts/1zDY7cJ7bA84JxxNBWDxBdui2xE", cfg.ServiceAccountMrn)
	assert.Equal(t, "-----BEGIN PRIVATE KEY-----\nMIG2AgE....C0Dvs=\n-----END PRIVATE KEY-----\n", cfg.PrivateKey)
	assert.Equal(t, "-----BEGIN CERTIFICATE-----\nMIICV .. fis=\n-----END CERTIFICATE-----\n", cfg.Certificate)

	assert.Equal(t, 10, cfg.ScanInterval.Timer)
	assert.Equal(t, 20, cfg.ScanInterval.Splay)

}

// TestConfigParsingDottedKeys verifies that nested config values work both as a
// nested mapping and as dotted keys. cnquery sets viper's key delimiter to "\\"
// (see mql cli/config InitViperConfig), so viper does not expand dotted keys
// into nested maps. ReadConfig folds the dotted form back in for backward
// compatibility. These tests reproduce production by setting that delimiter.
func TestConfigParsingDottedKeys(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		wantTimer  int
		wantSplay  int
		wantMethod string
	}{
		{
			name:       "nested",
			data:       "scan_interval:\n  timer: 360\n  splay: 30\nauth:\n  method: wif\n",
			wantTimer:  360,
			wantSplay:  30,
			wantMethod: "wif",
		},
		{
			name:       "dotted",
			data:       "scan_interval.timer: 360\nscan_interval.splay: 30\nauth.method: wif\n",
			wantTimer:  360,
			wantSplay:  30,
			wantMethod: "wif",
		},
		{
			name:      "dotted partial timer only",
			data:      "scan_interval.timer: 360\n",
			wantTimer: 360,
			wantSplay: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			// mirror mql's InitViperConfig: disable the "." key delimiter
			viper.SetOptions(viper.KeyDelimiter("\\"))
			viper.SetConfigType("yaml")
			require.NoError(t, viper.ReadConfig(strings.NewReader(tc.data)))

			cfg, err := ReadConfig()
			require.NoError(t, err)

			require.NotNil(t, cfg.ScanInterval)
			assert.Equal(t, tc.wantTimer, cfg.ScanInterval.Timer)
			assert.Equal(t, tc.wantSplay, cfg.ScanInterval.Splay)

			if tc.wantMethod != "" {
				require.NotNil(t, cfg.Authentication)
				assert.Equal(t, tc.wantMethod, cfg.Authentication.Method)
			}
		})
	}
}

// TestConfigParsingDottedMapKeysNotFolded locks in the boundary of the dotted-key
// compatibility shim: it must NOT fold dotted keys into map fields like
// annotations/labels. The "\\" key delimiter exists precisely so map keys can
// contain dots (e.g. "kubernetes.io/role"), so a dotted top-level form such as
// `annotations.foo: bar` is intentionally treated as a literal key, not a map
// entry. If someone later extends the shim to maps, this test should fail.
func TestConfigParsingDottedMapKeysNotFolded(t *testing.T) {
	viper.Reset()
	viper.SetOptions(viper.KeyDelimiter("\\"))
	viper.SetConfigType("yaml")
	data := "annotations.foo: bar\nlabels.env: prod\n"
	require.NoError(t, viper.ReadConfig(strings.NewReader(data)))

	cfg, err := ReadConfig()
	require.NoError(t, err)

	// dotted map keys are not folded into the nested maps
	assert.Empty(t, cfg.Annotations)
	assert.Empty(t, cfg.Labels)

	// the nested map form, with a dotted key, works as intended
	viper.Reset()
	viper.SetOptions(viper.KeyDelimiter("\\"))
	viper.SetConfigType("yaml")
	nested := "annotations:\n  kubernetes.io/role: db\n"
	require.NoError(t, viper.ReadConfig(strings.NewReader(nested)))

	cfg, err = ReadConfig()
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"kubernetes.io/role": "db"}, cfg.Annotations)
}
