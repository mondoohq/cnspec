package upstream

import (
	"context"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/gql"
	policy "go.mondoo.com/cnspec/v9/policy"
	mondoogql "go.mondoo.com/mondoo-go"
)

type PageInfo struct {
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
}

func SearchPolicy(ctx context.Context, c *gql.MondooClient, scopeMrn string, activeOnly bool) ([]*policy.Policy, error) {
	var m struct {
		Content struct {
			TotalCount int `json:"totalCount"`
			Edges      []struct {
				Cursor string `json:"cursor"`
				Node   struct {
					Policy struct {
						Mrn  string
						Name string
					} `graphql:"... on Policy"`
				} `json:"node"`
			} `json:"edges"`
			PageInfo PageInfo `json:"pageInfo"`
		} `graphql:"content(input: $input)"`
	}

	err := c.Query(ctx, &m, map[string]interface{}{
		"input": mondoogql.ContentSearchInput{
			ScopeMrn:     mondoogql.String(scopeMrn),
			CatalogType:  mondoogql.CatalogType("POLICY"),
			AssignedOnly: mondoogql.NewBooleanPtr(mondoogql.Boolean(activeOnly)),
		},
	})
	if err != nil {
		return nil, err
	}

	policies := make([]*policy.Policy, 0, len(m.Content.Edges))
	for _, edge := range m.Content.Edges {
		policies = append(policies, &policy.Policy{
			Mrn:  edge.Node.Policy.Mrn,
			Name: edge.Node.Policy.Name,
		})
	}
	return policies, nil
}
