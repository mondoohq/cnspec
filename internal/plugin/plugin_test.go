//go:build debugtest
// +build debugtest

package plugin_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/motor/asset"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/shared/proto"
	"go.mondoo.com/cnspec/internal/plugin"
)

func TestPlugin(t *testing.T) {
	inventory := &v1.Inventory{}
	inventory.AddAssets(&asset.Asset{
		Connections: []*providers.Config{{
			Backend: providers.ProviderType_LOCAL_OS,
			Options: map[string]string{},
		}},
	})

	err := plugin.RunQuery(&proto.RunQueryConfig{
		Command:   "mondoo.version",
		Features:  cnquery.DefaultFeatures,
		Inventory: inventory,
	})
	assert.NoError(t, err)
}
