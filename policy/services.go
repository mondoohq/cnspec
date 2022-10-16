package policy

import (
	"context"
	"net/http"

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
	DataLake  DataLake
	Upstream  *Services
	Incognito bool
}

// NewLocalServices initializes a reasonably configured local services struct
func NewLocalServices(datalake DataLake, uuid string) *LocalServices {
	return &LocalServices{
		DataLake:  datalake,
		Upstream:  nil,
		Incognito: false,
	}
}

// NewRemoteServices initializes a services struct with a remote endpoint
func NewRemoteServices(addr string, auth []ranger.ClientPlugin) (*Services, error) {
	client := ranger.DefaultHttpClient()
	// restrict parallel upstream connections to two connections
	client.Transport = NewMaxParallelConnTransport(client.Transport, 2)

	policyHub, err := NewPolicyHubClient(addr, client, auth...)
	if err != nil {
		return nil, err
	}

	policyResolver, err := NewPolicyResolverClient(addr, client, auth...)
	if err != nil {
		return nil, err
	}

	return &Services{
		PolicyHub:      policyHub,
		PolicyResolver: policyResolver,
	}, nil
}

// MaxParallelConnHTTPTransport restricts the parallel connections that the client is doing upstream.
// This has many advantages:
// - we do not run into max ulimit issues because of parallel execution
// - we do not ddos our server in case something is wrong upstream
// - implementing this as http.RoundTripper has the advantage that the http timeout still applies and calls are canceled properly on the client-side
type MaxParallelConnHTTPTransport struct {
	transport     http.RoundTripper
	parallelConns *semaphore.Weighted
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *MaxParallelConnHTTPTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	err := t.parallelConns.Acquire(r.Context(), 1)
	if err != nil {
		return nil, err
	}
	defer t.parallelConns.Release(1)
	return t.transport.RoundTrip(r)
}

// NewMaxParallelConnTransport creates a transport with parallel HTTP connections
func NewMaxParallelConnTransport(transport http.RoundTripper, parallel int64) *MaxParallelConnHTTPTransport {
	return &MaxParallelConnHTTPTransport{
		transport:     transport,
		parallelConns: semaphore.NewWeighted(parallel),
	}
}
