// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package upstream

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.mondoo.com/cnspec/v11/policy"
	mondoogql "go.mondoo.com/mondoo-go"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/gql"
)

type UpstreamFramework struct {
	policy.Framework

	State mondoogql.ComplianceFrameworkState
}

func ListFrameworks(ctx context.Context, c *gql.MondooClient, scopeMrn string, state *mondoogql.ComplianceFrameworkState) ([]*UpstreamFramework, error) {
	var q struct {
		Frameworks []struct {
			Mrn     string
			Name    string
			Version string
			State   mondoogql.ComplianceFrameworkState
		} `graphql:"complianceFrameworks(input: $input)"`
	}
	err := c.Query(ctx, &q, map[string]any{
		"input": mondoogql.ComplianceFrameworksInput{
			ScopeMrn: mondoogql.String(scopeMrn),
			State:    state,
		},
	})
	if err != nil {
		return nil, err
	}

	frameworks := make([]*UpstreamFramework, len(q.Frameworks))
	for i, f := range q.Frameworks {
		frameworks[i] = &UpstreamFramework{
			Framework: policy.Framework{
				Mrn:     f.Mrn,
				Name:    f.Name,
				Version: f.Version,
			},
			State: f.State,
		}
	}

	return frameworks, nil
}

func MutateFrameworkState(ctx context.Context, c *gql.MondooClient, mrn, scopeMrn string, action mondoogql.ComplianceFrameworkMutationAction) (bool, error) {
	var q struct {
		Mutation bool `graphql:"applyFrameworkMutation(input: $input)"`
	}
	err := c.Mutate(ctx, &q, mondoogql.ComplianceFrameworkMutationInput{
		FrameworkMrn: mondoogql.String(mrn),
		ScopeMrn:     mondoogql.String(scopeMrn),
		Action:       action,
	}, nil)
	return q.Mutation, err
}

func DownloadFramework(ctx context.Context, c *gql.MondooClient, mrn, scopeMrn string) (string, error) {
	var q struct {
		Download struct {
			Yaml string
		} `graphql:"downloadFramework(input: $input)"`
	}
	err := c.Query(ctx, &q, map[string]any{
		"input": mondoogql.DownloadFrameworkInput{
			Mrn:      mondoogql.String(mrn),
			ScopeMrn: mondoogql.String(scopeMrn),
		},
	})
	if err != nil {
		return "", err
	}

	return q.Download.Yaml, nil
}

func UploadFramework(ctx context.Context, c *gql.MondooClient, yaml []byte, spaceMrn string) (bool, error) {
	var q struct {
		Result bool `graphql:"uploadFramework(input: $input)"`
	}

	data := base64.StdEncoding.EncodeToString(yaml)
	err := c.Mutate(ctx, &q, mondoogql.UploadFrameworkInput{
		SpaceMrn: mondoogql.String(spaceMrn),
		Dataurl:  mondoogql.String(fmt.Sprintf("data:application/octet-stream;base64,%s", data)),
	}, nil)
	return q.Result, err
}
