// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package credentialcheck

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

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
