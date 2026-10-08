// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream"
)

func testPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func TestVerifyServiceAccount(t *testing.T) {
	const source = "/etc/opt/mondoo/mondoo.yml"

	t.Run("no service account", func(t *testing.T) {
		assert.NoError(t, verifyServiceAccount(nil, source))
	})

	t.Run("usable private key", func(t *testing.T) {
		creds := &upstream.ServiceAccountCredentials{
			Mrn:        "//agents.api.mondoo.app/spaces/test/serviceaccounts/test",
			PrivateKey: testPrivateKeyPEM(t),
		}
		assert.NoError(t, verifyServiceAccount(creds, source))
	})

	t.Run("static token is not parsed", func(t *testing.T) {
		creds := &upstream.ServiceAccountCredentials{Token: "token"}
		assert.NoError(t, verifyServiceAccount(creds, source))
	})

	for name, key := range map[string]string{
		"truncated private key": "x",
		"empty private key":     "",
		"not a PEM block":       "-----BEGIN PRIVATE KEY-----\nnot base64\n-----END PRIVATE KEY-----\n",
	} {
		t.Run(name, func(t *testing.T) {
			creds := &upstream.ServiceAccountCredentials{
				Mrn:        "//agents.api.mondoo.app/spaces/test/serviceaccounts/test",
				PrivateKey: key,
			}
			err := verifyServiceAccount(creds, source)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "the Mondoo service account in "+source+" can't be used")
			assert.Contains(t, err.Error(), "cnspec login")
			assert.Contains(t, err.Error(), "MONDOO_CONFIG_PATH")
			// the error must never echo key material
			if len(key) > 1 {
				assert.NotContains(t, err.Error(), key)
			}
		})
	}

	t.Run("unknown source", func(t *testing.T) {
		err := verifyServiceAccount(&upstream.ServiceAccountCredentials{PrivateKey: "x"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "the Mondoo service account in the Mondoo configuration can't be used")
	})
}

func TestBomFailuresError(t *testing.T) {
	names := map[string]string{"//assets/1": "host-1"}

	t.Run("all assets succeeded", func(t *testing.T) {
		assert.NoError(t, bomFailuresError("SBOM", nil, nil, names, 1))
	})

	t.Run("nothing generated and nothing failed", func(t *testing.T) {
		assert.NoError(t, bomFailuresError("SBOM", nil, nil, nil, 0))
	})

	t.Run("failed asset", func(t *testing.T) {
		err := bomFailuresError("SBOM",
			[]bomFailure{{Asset: "host-1", Reason: "no data points found"}},
			map[string]string{"//assets/1": "asset doesn't support any policies"},
			names, 0)
		require.Error(t, err)
		assert.True(t, errors.Is(err, errBomGenerationFailed))
		assert.Contains(t, err.Error(), "could not generate the SBOM")
		assert.Contains(t, err.Error(), `asset "host-1": no data points found`)
		assert.Contains(t, err.Error(), `asset "host-1": scan error: asset doesn't support any policies`)
	})

	t.Run("partial failure still fails", func(t *testing.T) {
		err := bomFailuresError("AIBOM",
			[]bomFailure{{Asset: "host-2", Reason: "no data points found"}},
			nil, names, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not generate the AIBOM")
	})

	t.Run("scan errors without any generated BOM", func(t *testing.T) {
		err := bomFailuresError("SBOM", nil,
			map[string]string{"//assets/unknown": "connection refused"}, names, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `asset "//assets/unknown": scan error: connection refused`)
	})
}

func TestAssetNamesByMrn(t *testing.T) {
	assert.Empty(t, assetNamesByMrn(nil))
	report := &policy.ReportCollection{Assets: map[string]*inventory.Asset{
		"//assets/1": {Name: "host-1"},
	}}
	assert.Equal(t, map[string]string{"//assets/1": "host-1"}, assetNamesByMrn(report))
}
