// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gql

import (
	"encoding/json"
	"net/http"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	mondoogql "go.mondoo.com/mondoo-go"
	"go.mondoo.com/mondoo-go/option"
)

type MondooClient struct {
	*mondoogql.Client
}

// NewClient creates a new GraphQL client for the Mondoo API
// provide the http client used for rpc, to also pass in the proxy settings
func NewClient(upstream upstream.UpstreamConfig, httpClient *http.Client) (*MondooClient, error) {
	gqlEndpoint := upstream.ApiEndpoint + "/query"
	creds, err := json.Marshal(upstream.Creds)
	if err != nil {
		return nil, err
	}
	// Initialize the client
	mondooClient, err := mondoogql.NewClient(
		option.WithEndpoint(gqlEndpoint),
		option.WithHTTPClient(httpClient),
		option.WithServiceAccount(creds),
	)
	if err != nil {
		return nil, err
	}

	return &MondooClient{mondooClient}, nil
}
