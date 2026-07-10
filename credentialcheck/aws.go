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

// defaultAWSRegion is used when no region option is set on the connection.
const defaultAWSRegion = "us-east-1"

// awsConfigFromConfig builds an aws.Config from an inventory connection's
// options and credentials, without making any network calls. It supports:
//
//   - static access key / secret key (+ optional session token), from a
//     "access-key-id" option paired with a credential's secret
//   - assume-role, from a "role" (role ARN) option and an optional
//     "external-id" option
//
// The region defaults to "us-east-1" when no "region" option is set.
func awsConfigFromConfig(ctx context.Context, conf *inventory.Config) (aws.Config, error) {
	options := conf.GetOptions()

	region := options["region"]
	if region == "" {
		region = defaultAWSRegion
	}

	loadOpts := []func(*config.LoadOptions) error{config.WithRegion(region)}

	if roleArn := options["role"]; roleArn != "" {
		externalID := options["external-id"]
		base, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
		if err != nil {
			return aws.Config{}, err
		}
		stsClient := sts.NewFromConfig(base)
		provider := aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(stsClient, roleArn, func(o *stscreds.AssumeRoleOptions) {
			if externalID != "" {
				o.ExternalID = &externalID
			}
		}))
		loadOpts = append(loadOpts, config.WithCredentialsProvider(provider))
	} else if creds := conf.GetCredentials(); len(creds) > 0 {
		accessKeyID := options["access-key-id"]
		secretAccessKey := string(creds[0].GetSecret())
		sessionToken := options["session-token"]
		loadOpts = append(loadOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, sessionToken),
		))
	}

	return config.LoadDefaultConfig(ctx, loadOpts...)
}
