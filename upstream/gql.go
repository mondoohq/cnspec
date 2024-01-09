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

func SearchPolicy(
	ctx context.Context,
	c *gql.MondooClient,
	scopeMrn string,
	assingedOnly,
	includePublic,
	includePrivate *bool,
) ([]*policy.Policy, error) {
	var q struct {
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

	policies := make([]*policy.Policy, 0, len(q.Content.Edges))
	for _, edge := range q.Content.Edges {
		policies = append(policies, &policy.Policy{
			Mrn:  edge.Node.Policy.Mrn,
			Name: edge.Node.Policy.Name,
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
