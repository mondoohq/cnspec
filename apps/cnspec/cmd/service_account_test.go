// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
