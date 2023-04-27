package scan

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/upstream"
)

func TestFilterPreprocess(t *testing.T) {
	// given
	filters := []string{
		"namespace1/policy1",
		"namespace2/policy2",
		"//registry.mondoo.com/namespace/namespace3/policies/policy3",
	}

	// when
	preprocessed := preprocessPolicyFilters(filters)

	// then
	assert.Equal(t, []string{
		"//registry.mondoo.com/namespace/namespace1/policies/policy1",
		"//registry.mondoo.com/namespace/namespace2/policies/policy2",
		"//registry.mondoo.com/namespace/namespace3/policies/policy3",
	}, preprocessed)
}

func TestGetUpstreamConfig(t *testing.T) {
	t.Run("with upstream plugin", func(t *testing.T) {
		opts := []ScannerOption{
			WithUpstream("api-endpoint", "space-mrn", http.DefaultClient),
		}
		scanner := NewLocalScanner(opts...)
		_, err := scanner.getUpstreamConfig(false, &Job{})
		require.NoError(t, err)

		_, err = scanner.getUpstreamConfig(true, &Job{})
		require.NoError(t, err)
	})

	t.Run("with upstream plugin, but no http client", func(t *testing.T) {
		opts := []ScannerOption{
			WithUpstream("api-endpoint", "space-mrn", nil),
		}
		scanner := NewLocalScanner(opts...)
		_, err := scanner.getUpstreamConfig(false, &Job{})
		require.NoError(t, err)

		_, err = scanner.getUpstreamConfig(true, &Job{})
		require.NoError(t, err)
	})

	t.Run("with job creds", func(t *testing.T) {
		opts := []ScannerOption{
			AllowJobCredentials(),
		}

		pk, err := os.ReadFile("../testdata/private-key.p8")
		require.NoError(t, err)

		cert, err := os.ReadFile("../testdata/cert.pem")
		require.NoError(t, err)

		job := &Job{
			Inventory: &v1.Inventory{
				Spec: &v1.InventorySpec{
					UpstreamCredentials: &upstream.ServiceAccountCredentials{
						ApiEndpoint: "api",
						ParentMrn:   "space-mrn",
						PrivateKey:  string(pk),
						Certificate: string(cert),
					},
				},
			},
		}
		scanner := NewLocalScanner(opts...)
		_, err = scanner.getUpstreamConfig(false, job)
		require.NoError(t, err)

		_, err = scanner.getUpstreamConfig(true, &Job{})
		require.NoError(t, err)
	})
}
