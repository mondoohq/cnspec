// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gql

import (
	"context"

	mondoogql "go.mondoo.com/mondoo-go"
)

type ContentSearchResponse struct {
	TotalCount int          `json:"totalCount"`
	Edges      []PolicyEdge `json:"edges"`
	PageInfo   PageInfo     `json:"pageInfo"`
}

type PolicyEdge struct {
	Cursor string     `json:"cursor"`
	Node   PolicyNode `json:"node"`
}

type PolicyNode struct {
	// The Policy struct is embedded to match the inline fragment ... on Policy.
	// We assume that all these fields are part of the Policy type.
	*Policy `graphql:"... on Policy"`
}

// Policy contains fields that are expected to be part of the Policy type
// within the GraphQL schema.
type Policy struct {
	UID         *string      `json:"uid"`
	MRN         string       `json:"mrn"`
	Name        string       `json:"name"`
	Action      *string      `json:"action"`
	Version     string       `json:"version"`
	Summary     *string      `json:"summary"`
	Docs        string       `json:"docs"`
	Authors     []Author     `json:"authors"`
	Category    string       `json:"category"`
	TrustLevel  string       `json:"trustLevel"`
	Access      string       `json:"access"`
	Statistics  Statistics   `json:"statistics"`
	CertifiedBy *[]Certifier `json:"certifiedBy"`
	Featured    bool         `json:"featured"`
	Assigned    bool         `json:"assigned"`
	// You can add additional fields here that are part of the Policy type.
}

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Statistics struct {
	Checks   int `json:"checks"`
	Queries  int `json:"queries"`
	Policies int `json:"policies"`
}

type Certifier struct {
	Name string `json:"name"` // This may need to be *string if null values are possible.
}

type PageInfo struct {
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
}

func (c *MondooClient) SearchPolicy(mrn string, activeOnly bool, catalogType string) (*ContentSearchResponse, error) {
	var m struct {
		Content struct {
			TotalCount int `json:"totalCount"`
			Edges      []struct {
				Cursor string `json:"cursor"`
				Node   struct {
					// Embed the Policy struct directly, this represents the ... on Policy fragment
					Policy `graphql:"... on Policy"`
				} `json:"node"`
			} `json:"edges"`
			PageInfo PageInfo `json:"pageInfo"`
		} `graphql:"content(input: $input)"`
	}

	err := c.Query(context.Background(), &m, map[string]interface{}{
		"input": mondoogql.ContentSearchInput{
			ScopeMrn:     mondoogql.String(mrn),
			CatalogType:  mondoogql.CatalogType(catalogType),
			AssignedOnly: mondoogql.NewBooleanPtr(mondoogql.Boolean(activeOnly)),
			Categories:   &[]mondoogql.String{"security"},
		},
	})
	if err != nil {
		return nil, err
	}

	response := &ContentSearchResponse{
		TotalCount: m.Content.TotalCount,
		PageInfo:   m.Content.PageInfo,
		Edges:      make([]PolicyEdge, 0, len(m.Content.Edges)),
	}

	for _, edge := range m.Content.Edges {
		policyNode := Policy{
			UID:         edge.Node.UID,
			MRN:         edge.Node.MRN,
			Name:        edge.Node.Name,
			Action:      edge.Node.Action,
			Version:     edge.Node.Version,
			Summary:     edge.Node.Summary,
			Docs:        edge.Node.Docs,
			Authors:     edge.Node.Authors,
			Category:    edge.Node.Category,
			TrustLevel:  edge.Node.TrustLevel,
			Access:      edge.Node.Access,
			Statistics:  edge.Node.Statistics,
			CertifiedBy: edge.Node.CertifiedBy,
			Featured:    edge.Node.Featured,
			Assigned:    edge.Node.Assigned,
		}

		response.Edges = append(response.Edges, PolicyEdge{
			Cursor: edge.Cursor,
			Node:   PolicyNode{Policy: &policyNode},
		})
	}
	return response, nil
}
