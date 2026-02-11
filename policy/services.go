// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"net/http"
	"time"

	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"go.mondoo.com/ranger-rpc"
	"golang.org/x/sync/semaphore"
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
	DataLake    DataLake
	Upstream    *Services
	Incognito   bool
	Runtime     llx.Runtime
	NowProvider func() time.Time
}

// NewLocalServices initializes a reasonably configured local services struct
func NewLocalServices(datalake DataLake, runtime llx.Runtime) *LocalServices {
	return &LocalServices{
		DataLake:    datalake,
		Upstream:    nil,
		Incognito:   false,
		Runtime:     runtime,
		NowProvider: time.Now,
	}
}

// NewRemoteServices initializes a services struct with a remote endpoint
func NewRemoteServices(addr string, auth []ranger.ClientPlugin, httpClient *http.Client) (*Services, error) {
	if httpClient == nil {
		httpClient = ranger.DefaultHttpClient()
	}

	// restrict parallel upstream connections to two connections
	httpClient.Transport = newMaxParallelConnTransport(httpClient.Transport, 2)

	policyHub, err := NewPolicyHubClient(addr, httpClient, auth...)
	if err != nil {
		return nil, err
	}

	var policyResolver PolicyResolver
	policyResolver, err = NewPolicyResolverClient(addr, httpClient, auth...)
	if err != nil {
		return nil, err
	}

	return &Services{
		PolicyHub:      policyHub,
		PolicyResolver: policyResolver,
	}, nil
}

func (l *LocalServices) Schema() resources.ResourcesSchema {
	return l.Runtime.Schema()
}

func (l *LocalServices) NewCompilerConfig() mqlc.CompilerConfig {
	return mqlc.NewConfig(l.Schema(), mql.DefaultFeatures)
}

// maxParallelConnHTTPTransport restricts the parallel connections upstream.
type maxParallelConnHTTPTransport struct {
	transport     http.RoundTripper
	parallelConns *semaphore.Weighted
}

func (t *maxParallelConnHTTPTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	err := t.parallelConns.Acquire(r.Context(), 1)
	if err != nil {
		return nil, err
	}
	defer t.parallelConns.Release(1)
	return t.transport.RoundTrip(r)
}

func newMaxParallelConnTransport(transport http.RoundTripper, parallel int64) *maxParallelConnHTTPTransport {
	return &maxParallelConnHTTPTransport{
		transport:     transport,
		parallelConns: semaphore.NewWeighted(parallel),
	}
}

type NoStoreResults struct {
	PolicyResolver
}

func (n *NoStoreResults) StoreResults(context.Context, *StoreResultsReq) (*Empty, error) {
	return globalEmpty, nil
}
