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
`

	viper.SetConfigType("yaml")
	viper.ReadConfig(strings.NewReader(data))

	cfg, err := ReadConfig()
	require.NoError(t, err)
	assert.Equal(t, "//agents.api.mondoo.app/spaces/musing-saha-952142/agents/1zDY7auR20SgrFfiGUT5qZWx6mE", cfg.AgentMrn)
	assert.Equal(t, "//agents.api.mondoo.app/spaces/musing-saha-952142/serviceaccounts/1zDY7cJ7bA84JxxNBWDxBdui2xE", cfg.ServiceAccountMrn)
	assert.Equal(t, "-----BEGIN PRIVATE KEY-----\nMIG2AgE....C0Dvs=\n-----END PRIVATE KEY-----\n", cfg.PrivateKey)
	assert.Equal(t, "-----BEGIN CERTIFICATE-----\nMIICV .. fis=\n-----END CERTIFICATE-----\n", cfg.Certificate)
}
