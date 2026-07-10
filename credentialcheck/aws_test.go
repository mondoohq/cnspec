// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package credentialcheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

func TestAwsConfigFromConfig_StaticCreds(t *testing.T) {
	conf := &inventory.Config{
		Type:    "aws",
		Options: map[string]string{"region": "eu-central-1", "access-key-id": "AKIAEXAMPLE"},
		Credentials: []*vault.Credential{{
			Type:   vault.CredentialType_password,
			Secret: []byte("secret-key"),
		}},
	}
	cfg, err := awsConfigFromConfig(context.Background(), conf)
	require.NoError(t, err)
	require.Equal(t, "eu-central-1", cfg.Region)

	creds, err := cfg.Credentials.Retrieve(context.Background())
	require.NoError(t, err)
	require.Equal(t, "AKIAEXAMPLE", creds.AccessKeyID)
	require.Equal(t, "secret-key", creds.SecretAccessKey)
}

func TestAwsConfigFromConfig_StaticCredsWithSessionToken(t *testing.T) {
	conf := &inventory.Config{
		Type: "aws",
		Options: map[string]string{
			"region":        "us-west-2",
			"access-key-id": "AKIAEXAMPLE",
			"session-token": "session-token-value",
		},
		Credentials: []*vault.Credential{{
			Type:   vault.CredentialType_password,
			Secret: []byte("secret-key"),
		}},
	}
	cfg, err := awsConfigFromConfig(context.Background(), conf)
	require.NoError(t, err)

	creds, err := cfg.Credentials.Retrieve(context.Background())
	require.NoError(t, err)
	require.Equal(t, "AKIAEXAMPLE", creds.AccessKeyID)
	require.Equal(t, "secret-key", creds.SecretAccessKey)
	require.Equal(t, "session-token-value", creds.SessionToken)
}

func TestAwsConfigFromConfig_DefaultRegion(t *testing.T) {
	conf := &inventory.Config{
		Type: "aws",
		Credentials: []*vault.Credential{{
			Type:   vault.CredentialType_password,
			Secret: []byte("s"),
		}},
	}
	cfg, err := awsConfigFromConfig(context.Background(), conf)
	require.NoError(t, err)
	require.Equal(t, "us-east-1", cfg.Region)
}

func TestAwsConfigFromConfig_AssumeRole(t *testing.T) {
	conf := &inventory.Config{
		Type: "aws",
		Options: map[string]string{
			"region":      "eu-central-1",
			"role":        "arn:aws:iam::123456789012:role/example-role",
			"external-id": "some-external-id",
		},
	}
	cfg, err := awsConfigFromConfig(context.Background(), conf)
	require.NoError(t, err)
	require.Equal(t, "eu-central-1", cfg.Region)
	// The assume-role credentials provider only contacts STS when Retrieve is
	// called, so we only assert it was set up, without ever calling Retrieve.
	require.NotNil(t, cfg.Credentials)
}

func TestAwsConfigFromConfig_NoCredentials(t *testing.T) {
	conf := &inventory.Config{Type: "aws"}
	cfg, err := awsConfigFromConfig(context.Background(), conf)
	require.NoError(t, err)
	require.Equal(t, "us-east-1", cfg.Region)
}
