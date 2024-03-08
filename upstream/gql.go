// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package upstream

import (
	"context"
	"encoding/json"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/mondoo-go/option"
	"net/http"

	policy "go.mondoo.com/cnspec/v10/policy"
	mondoogql "go.mondoo.com/mondoo-go"
)

type MondooClient struct {
	*mondoogql.Client
}

// NewClient creates a new GraphQL client for the Mondoo API
// provide the http client used for rpc, to also pass in the proxy settings
func NewClient(upstream *upstream.UpstreamConfig, httpClient *http.Client) (*MondooClient, error) {
	gqlEndpoint := upstream.ApiEndpoint + "/query"
	creds, err := json.Marshal(upstream.Creds)
	if err != nil {
		return nil, err
	}
	// Initialize the client
	gqlClient, err := mondoogql.NewClient(
		option.WithEndpoint(gqlEndpoint),
		option.WithHTTPClient(httpClient),
		option.WithServiceAccount(creds),
	)
	if err != nil {
		return nil, err
	}

	return &MondooClient{gqlClient}, nil
}

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

func (c *MondooClient) SearchPolicy(
	ctx context.Context,
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

func (c *MondooClient) GetSpace(ctx context.Context, mrn string) (*Space, error) {
	var q struct {
		Space Space `graphql:"space(mrn: $mrn)"`
	}
	err := c.Query(ctx, &q, map[string]interface{}{"mrn": mondoogql.String(mrn)})
	if err != nil {
		return nil, err
	}
	return &q.Space, nil
}
