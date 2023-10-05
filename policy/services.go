// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"net/http"

	"go.mondoo.com/cnquery/v9/explorer"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/ranger-rpc"
)

type ResolvedPolicyVersion string

const (
	V2Code ResolvedPolicyVersion = "v2"
)

var globalEmpty = &Empty{}

// Library is a subset of the DataLake focused on methods around policy and query existence
type Library interface {
	// QueryExists checks if the given MRN exists
	QueryExists(ctx context.Context, mrn string) (bool, error)
	// PolicyExists checks if the given MRN exists
	PolicyExists(ctx context.Context, mrn string) (bool, error)
}

type Services struct {
	PolicyHub
	PolicyResolver
}

// LocalServices is a bundle of all the services for handling policies.
// It has an optional upstream-handler embedded. If a local service does not
// yield results for a request, and the upstream handler is defined, it will
// be used instead.
type LocalServices struct {
	DataLake  DataLake
	Upstream  *Services
	Incognito bool
	runtime   llx.Runtime
}

// NewLocalServices initializes a reasonably configured local services struct
func NewLocalServices(datalake DataLake, uuid string, runtime llx.Runtime) *LocalServices {
	return &LocalServices{
		DataLake:  datalake,
		Upstream:  nil,
		Incognito: false,
		runtime:   runtime,
	}
}

// NewRemoteServices initializes a services struct with a remote endpoint
func NewRemoteServices(addr string, auth []ranger.ClientPlugin, httpClient *http.Client) (*Services, error) {
	if httpClient == nil {
		httpClient = ranger.DefaultHttpClient()
	}

	// restrict parallel upstream connections to two connections
	httpClient.Transport = explorer.NewMaxParallelConnTransport(httpClient.Transport, 2)

	policyHub, err := NewPolicyHubClient(addr, httpClient, auth...)
	if err != nil {
		return nil, err
	}

	policyResolver, err := NewPolicyResolverClient(addr, httpClient, auth...)
	if err != nil {
		return nil, err
	}

	return &Services{
		PolicyHub:      policyHub,
		PolicyResolver: policyResolver,
	}, nil
}

func (l *LocalServices) Schema() llx.Schema {
	return l.runtime.Schema()
}
