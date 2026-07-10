// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package credentialcheck

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

func TestValidate_UnsupportedProvider(t *testing.T) {
	res, err := Validate(context.Background(), &inventory.Config{Type: "gcp"})
	require.NoError(t, err)
	require.Equal(t, StateUnknown, res.State)
	require.Contains(t, res.Message, "not supported")
	require.Nil(t, res.ExpiresAt)
}

// TestValidate_AWS_AuthError_EndToEnd drives the public Validate dispatcher
// for an AWS config all the way through to its AuthError classification,
// with the STS identity check swapped out for a fake so the test never makes
// a real AWS call.
func TestValidate_AWS_AuthError_EndToEnd(t *testing.T) {
	original := stsIdentityCheck
	stsIdentityCheck = func(context.Context, aws.Config) error {
		return &smithy.GenericAPIError{Code: "InvalidClientTokenId"}
	}
	t.Cleanup(func() { stsIdentityCheck = original })

	conf := &inventory.Config{
		Type:    "aws",
		Options: map[string]string{"region": "eu-central-1", "access-key-id": "AKIAEXAMPLE"},
		Credentials: []*vault.Credential{{
			Type:   vault.CredentialType_password,
			Secret: []byte("secret-key"),
		}},
	}

	res, err := Validate(context.Background(), conf)
	require.NoError(t, err)
	require.Equal(t, StateAuthError, res.State)
	require.Contains(t, res.Message, "InvalidClientTokenId")
	require.Nil(t, res.ExpiresAt)
}

func TestState_String(t *testing.T) {
	require.Equal(t, "OK", StateOK.String())
	require.Equal(t, "AUTH_ERROR", StateAuthError.String())
	require.Equal(t, "UNKNOWN", StateUnknown.String())
}
