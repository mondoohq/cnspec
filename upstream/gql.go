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
	assignedOnly,
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
	if assignedOnly != nil {
		input.AssignedOnly = mondoogql.NewBooleanPtr(mondoogql.Boolean(*assignedOnly))
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
