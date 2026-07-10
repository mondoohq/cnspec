// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package credentialcheck

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

// clearAmbientAWSConfig resets every AWS SDK input that could resolve an
// ambient region or credentials (env vars, shared config/credentials files,
// EC2 instance metadata) so that tests exercising the "no region option"
// fallback path get a deterministic, empty ambient configuration instead of
// depending on whatever the local environment happens to contain.
func clearAmbientAWSConfig(t *testing.T) {
	t.Helper()
	nonExistent := filepath.Join(t.TempDir(), "does-not-exist")
	for _, key := range []string{
		"AWS_REGION",
		"AWS_DEFAULT_REGION",
		"AWS_PROFILE",
		"AWS_DEFAULT_PROFILE",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("AWS_CONFIG_FILE", nonExistent)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", nonExistent)
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
}

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
	// No "region" option and no ambient region: falls back to the default.
	clearAmbientAWSConfig(t)
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

func TestAwsConfigFromConfig_RegionOptionWinsOverAmbient(t *testing.T) {
	// An explicit "region" option must win even when an ambient region is
	// also present, e.g. via AWS_REGION.
	t.Setenv("AWS_REGION", "ap-southeast-2")
	conf := &inventory.Config{
		Type:    "aws",
		Options: map[string]string{"region": "eu-central-1"},
	}
	cfg, err := awsConfigFromConfig(context.Background(), conf)
	require.NoError(t, err)
	require.Equal(t, "eu-central-1", cfg.Region)
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
	clearAmbientAWSConfig(t)
	conf := &inventory.Config{Type: "aws"}
	cfg, err := awsConfigFromConfig(context.Background(), conf)
	require.NoError(t, err)
	require.Equal(t, "us-east-1", cfg.Region)
}

func TestAwsConfigFromConfig_RoleWinsOverStaticCreds(t *testing.T) {
	// When both an assume-role option and static credentials are present,
	// the assume-role provider must be selected deterministically. Role and
	// static credentials are mutually exclusive in the inventories this
	// package validates, but this documents the intentional precedence.
	conf := &inventory.Config{
		Type: "aws",
		Options: map[string]string{
			"region":        "eu-central-1",
			"role":          "arn:aws:iam::123456789012:role/example-role",
			"access-key-id": "AKIAEXAMPLE",
		},
		Credentials: []*vault.Credential{{
			Type:   vault.CredentialType_password,
			Secret: []byte("secret-key"),
		}},
	}
	cfg, err := awsConfigFromConfig(context.Background(), conf)
	require.NoError(t, err)

	// The assume-role provider is wrapped in an *aws.CredentialsCache; the
	// static provider is set directly, so the wrapper type alone
	// distinguishes which path was taken without ever calling Retrieve (and
	// thus without making an STS network call).
	_, isCache := cfg.Credentials.(*aws.CredentialsCache)
	require.True(t, isCache, "expected assume-role credentials provider to be selected")

	_, isStatic := cfg.Credentials.(credentials.StaticCredentialsProvider)
	require.False(t, isStatic, "static credentials provider must not be selected when a role is also set")
}

func TestClassifyAWSError(t *testing.T) {
	require.Equal(t, StateOK, classifyAWSError(nil))
	require.Equal(t, StateAuthError, classifyAWSError(&smithy.GenericAPIError{Code: "InvalidClientTokenId"}))
	require.Equal(t, StateAuthError, classifyAWSError(&smithy.GenericAPIError{Code: "SignatureDoesNotMatch"}))
	require.Equal(t, StateAuthError, classifyAWSError(&smithy.GenericAPIError{Code: "UnrecognizedClientException"}))
	require.Equal(t, StateAuthError, classifyAWSError(&smithy.GenericAPIError{Code: "AccessDenied"}))
	require.Equal(t, StateAuthError, classifyAWSError(&smithy.GenericAPIError{Code: "AccessDeniedException"}))
	require.Equal(t, StateAuthError, classifyAWSError(&smithy.GenericAPIError{Code: "ExpiredToken"}))
	require.Equal(t, StateAuthError, classifyAWSError(&smithy.GenericAPIError{Code: "ExpiredTokenException"}))
	require.Equal(t, StateUnknown, classifyAWSError(&smithy.GenericAPIError{Code: "ThrottlingException"}))
	require.Equal(t, StateUnknown, classifyAWSError(errors.New("dial tcp: i/o timeout")))
}

func TestValidateAWS_OK_AuthError_Unknown(t *testing.T) {
	conf := &inventory.Config{Type: "aws", Options: map[string]string{"region": "eu-central-1", "access-key-id": "AKIA"}, Credentials: []*vault.Credential{{
		Type: vault.CredentialType_password, Secret: []byte("s"),
	}}}

	res := validateAWSWith(context.Background(), conf, func(context.Context, aws.Config) error { return nil })
	require.Equal(t, StateOK, res.State)
	require.NotEmpty(t, res.Message)
	require.Nil(t, res.ExpiresAt)

	res = validateAWSWith(context.Background(), conf, func(context.Context, aws.Config) error {
		return &smithy.GenericAPIError{Code: "InvalidClientTokenId"}
	})
	require.Equal(t, StateAuthError, res.State)
	require.NotEmpty(t, res.Message)
	require.Nil(t, res.ExpiresAt)

	res = validateAWSWith(context.Background(), conf, func(context.Context, aws.Config) error {
		return errors.New("network down")
	})
	require.Equal(t, StateUnknown, res.State)
	require.NotEmpty(t, res.Message)
	require.Nil(t, res.ExpiresAt)
}

func TestValidateAWS_RetriesWithGovCloudRegionOnFailure(t *testing.T) {
	conf := &inventory.Config{Type: "aws", Options: map[string]string{"region": "eu-central-1", "access-key-id": "AKIA"}, Credentials: []*vault.Credential{{
		Type: vault.CredentialType_password, Secret: []byte("s"),
	}}}

	var regionsSeen []string
	res := validateAWSWith(context.Background(), conf, func(_ context.Context, cfg aws.Config) error {
		regionsSeen = append(regionsSeen, cfg.Region)
		if cfg.Region == "us-gov-west-1" {
			return nil
		}
		return errors.New("network down")
	})
	require.Equal(t, StateOK, res.State)
	require.Equal(t, []string{"eu-central-1", "us-gov-west-1"}, regionsSeen)
}

func TestValidateAWS_ReturnsFirstErrorWhenGovCloudFallbackAlsoFails(t *testing.T) {
	conf := &inventory.Config{Type: "aws", Options: map[string]string{"region": "eu-central-1", "access-key-id": "AKIA"}, Credentials: []*vault.Credential{{
		Type: vault.CredentialType_password, Secret: []byte("s"),
	}}}

	res := validateAWSWith(context.Background(), conf, func(_ context.Context, cfg aws.Config) error {
		if cfg.Region == "eu-central-1" {
			return &smithy.GenericAPIError{Code: "InvalidClientTokenId"}
		}
		return errors.New("gov cloud also down")
	})
	// classification must be based on the *original* error, mirroring Verify's
	// GovCloud fallback behavior in the AWS connection provider.
	require.Equal(t, StateAuthError, res.State)
}
