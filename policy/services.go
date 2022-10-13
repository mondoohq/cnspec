package policy

import "context"

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
