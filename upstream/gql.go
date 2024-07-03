// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package upstream

import (
	"context"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/gql"
	policy "go.mondoo.com/cnspec/v11/policy"
	mondoogql "go.mondoo.com/mondoo-go"
)

type PageInfo struct {
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
}

type UpstreamPolicy struct {
	policy.Policy
	TrustLevel mondoogql.TrustLevel
	Action     mondoogql.PolicyAction
	Assigned   bool
}

func SearchPolicy(
	ctx context.Context,
	c *gql.MondooClient,
	scopeMrn string,
	assingedOnly,
	includePublic,
	includePrivate *bool,
) ([]*UpstreamPolicy, error) {
	var q struct {
		Content struct {
			TotalCount int `json:"totalCount"`
			Edges      []struct {
				Cursor string `json:"cursor"`
				Node   struct {
					Policy struct {
						Mrn        string
						Name       string
						TrustLevel mondoogql.TrustLevel
						Action     mondoogql.PolicyAction
						Assigned   bool
					} `graphql:"... on Policy"`
				} `json:"node"`
			} `json:"edges"`
			PageInfo PageInfo `json:"pageInfo"`
		} `graphql:"content(input: $input)"`
	}

	input := mondoogql.ContentSearchInput{
		ScopeMrn:    mondoogql.String(scopeMrn),
		CatalogType: mondoogql.CatalogType("POLICY"),
	}
	if assingedOnly != nil {
		input.AssignedOnly = mondoogql.NewBooleanPtr(mondoogql.Boolean(*assingedOnly))
	}
	if includePublic != nil {
		input.IncludePublic = mondoogql.NewBooleanPtr(mondoogql.Boolean(*includePublic))
	}
	if includePrivate != nil {
		input.IncludePrivate = mondoogql.NewBooleanPtr(mondoogql.Boolean(*includePrivate))
	}
	err := c.Query(ctx, &q, map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return nil, err
	}

	policies := make([]*UpstreamPolicy, 0, len(q.Content.Edges))
	for _, edge := range q.Content.Edges {
		policies = append(policies, &UpstreamPolicy{
			Policy: policy.Policy{
				Mrn:  edge.Node.Policy.Mrn,
				Name: edge.Node.Policy.Name,
			},
			TrustLevel: edge.Node.Policy.TrustLevel,
			Action:     edge.Node.Policy.Action,
			Assigned:   edge.Node.Policy.Assigned,
		})
	}
	return policies, nil
}

type Space struct {
	Mrn  string
	Name string
}

func GetSpace(ctx context.Context, c *gql.MondooClient, mrn string) (*Space, error) {
	var q struct {
		Space Space `graphql:"space(mrn: $mrn)"`
	}
	err := c.Query(ctx, &q, map[string]interface{}{"mrn": mondoogql.String(mrn)})
	if err != nil {
		return nil, err
	}
	return &q.Space, nil
}

type Framework struct {
	Authors []struct {
		Name  string
		Email string
	} `graphql:"authors"`
	Name    string
	Mrn     string
	State   mondoogql.ComplianceFrameworkState
	Version string
}

func ListFrameworks(ctx context.Context, c *gql.MondooClient, scopeMrn string) ([]*Framework, error) {
	var q struct {
		Frameworks []*Framework `graphql:"complianceFrameworks(input: $input)"`
	}
	err := c.Query(ctx, &q, map[string]any{
		"input": mondoogql.ComplianceFrameworksInput{
			ScopeMrn: mondoogql.String(scopeMrn),
		},
	})
	if err != nil {
		return nil, err
	}

	return q.Frameworks, nil
}

func MutateFrameworkState(ctx context.Context, c *gql.MondooClient, mrn, scopeMrn string, action mondoogql.ComplianceFrameworkMutationAction) error {
	var q struct {
		Mutation bool `graphql:"applyFrameworkMutation(input: $input)"`
	}
	return c.Mutate(ctx, &q, mondoogql.ComplianceFrameworkMutationInput{
		FrameworkMrn: mondoogql.String(mrn),
		ScopeMrn:     mondoogql.String(scopeMrn),
		Action:       action,
	}, nil)
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
