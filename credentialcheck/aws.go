// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package credentialcheck

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// govCloudFallbackRegion is retried when the identity check fails in the
// configured (or default) region. Some AWS credentials are only valid in the
// AWS GovCloud partition, which uses a disjoint region namespace.
const govCloudFallbackRegion = "us-gov-west-1"

// authErrorCodes are the AWS API error codes that indicate the credential
// itself was rejected (bad access key, bad signature, expired token, no
// permission), as opposed to a transient or unclassifiable failure.
var authErrorCodes = map[string]bool{
	"InvalidClientTokenId":        true,
	"SignatureDoesNotMatch":       true,
	"UnrecognizedClientException": true,
	"AccessDenied":                true,
	"AccessDeniedException":       true,
	"ExpiredToken":                true,
	"ExpiredTokenException":       true,
}

// classifyAWSError maps an error returned by an AWS API call to a validation
// State. A nil error means the call succeeded (StateOK). An AWS API error
// whose code identifies it as a credential/authorization problem maps to
// StateAuthError. Anything else (network errors, timeouts, throttling,
// service outages, ...) maps to StateUnknown, since it does not tell us
// whether the credential itself is valid.
func classifyAWSError(err error) State {
	if err == nil {
		return StateOK
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && authErrorCodes[apiErr.ErrorCode()] {
		return StateAuthError
	}
	return StateUnknown
}

// stsIdentityFunc performs the actual identity check against AWS STS using
// the given config. It exists so tests can inject a fake and never make a
// real network call.
type stsIdentityFunc func(ctx context.Context, cfg aws.Config) error

// realSTSIdentityCheck calls STS GetCallerIdentity, the cheapest AWS API call
// that proves a credential is valid and returns a definitive error otherwise.
func realSTSIdentityCheck(ctx context.Context, cfg aws.Config) error {
	_, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	return err
}

// stsIdentityCheck is the STS identity check used by validateAWS. It is a
// package variable, rather than a direct call, purely so tests can swap it
// out to exercise the full Validate dispatch (including the AWS branch) end
// to end without making a real network call; production code never
// reassigns it.
var stsIdentityCheck stsIdentityFunc = realSTSIdentityCheck

// validateAWS builds an aws.Config from conf and confirms its credentials
// authenticate by calling STS GetCallerIdentity.
func validateAWS(ctx context.Context, conf *inventory.Config) Result {
	return validateAWSWith(ctx, conf, stsIdentityCheck)
}

// validateAWSWith is the test seam for validateAWS: it performs the identity
// check via the given stsIdentityFunc instead of always calling real AWS.
func validateAWSWith(ctx context.Context, conf *inventory.Config, check stsIdentityFunc) Result {
	cfg, err := awsConfigFromConfig(ctx, conf)
	if err != nil {
		return Result{State: StateUnknown, Message: fmt.Sprintf("could not validate credentials: %v", err)}
	}

	err = check(ctx, cfg)
	if err != nil {
		// Some credentials are only valid in the AWS GovCloud partition. Mirror
		// the AWS connection provider's fallback: retry once in that region
		// before giving up, but classify based on the original error if the
		// fallback also fails.
		firstErr := err
		govCfg := cfg.Copy()
		govCfg.Region = govCloudFallbackRegion
		if govErr := check(ctx, govCfg); govErr == nil {
			err = nil
		} else {
			err = firstErr
		}
	}

	switch state := classifyAWSError(err); state {
	case StateOK:
		return Result{State: StateOK, Message: "credentials valid"}
	case StateAuthError:
		var apiErr smithy.APIError
		errors.As(err, &apiErr)
		return Result{State: StateAuthError, Message: fmt.Sprintf("authentication rejected: %s", apiErr.ErrorCode())}
	default:
		return Result{State: StateUnknown, Message: fmt.Sprintf("could not validate credentials: %v", err)}
	}
}

// defaultAWSRegion is the last-resort region used when neither a "region"
// option nor an ambient region (env var, shared config profile, etc.) can be
// resolved.
const defaultAWSRegion = "us-east-1"

// awsConfigFromConfig builds an aws.Config from an inventory connection's
// options and credentials, without making any network calls. It supports:
//
//   - static access key / secret key (+ optional session token), from a
//     "access-key-id" option paired with a credential's secret
//   - assume-role, from a "role" (role ARN) option and an optional
//     "external-id" option
//
// Region resolution mirrors the AWS connection provider: an explicit
// "region" option always wins; otherwise the ambient region resolved by
// config.LoadDefaultConfig (env vars, shared config profile, etc.) is used,
// and only if that is also empty do we fall back to defaultAWSRegion.
func awsConfigFromConfig(ctx context.Context, conf *inventory.Config) (aws.Config, error) {
	options := conf.GetOptions()

	var loadOpts []func(*config.LoadOptions) error
	if region := options["region"]; region != "" {
		loadOpts = append(loadOpts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return aws.Config{}, err
	}
	if cfg.Region == "" {
		// No region option and no ambient region could be resolved; fall
		// back to the same default used by the AWS connection provider.
		cfg.Region = defaultAWSRegion
	}

	// Role-based and static credentials are mutually exclusive in the
	// inventories this package validates, so this precedence is academic in
	// practice. If both were somehow set, the assume-role provider takes
	// precedence.
	if roleArn := options["role"]; roleArn != "" {
		externalID := options["external-id"]
		stsClient := sts.NewFromConfig(cfg)
		cfg.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(stsClient, roleArn, func(o *stscreds.AssumeRoleOptions) {
			if externalID != "" {
				o.ExternalID = &externalID
			}
		}))
	} else if creds := conf.GetCredentials(); len(creds) > 0 {
		accessKeyID := options["access-key-id"]
		secretAccessKey := string(creds[0].GetSecret())
		sessionToken := options["session-token"]
		cfg.Credentials = credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, sessionToken)
	}

	return cfg, nil
}
