package policy

import "context"

// Library is a subset of the DataLake focused on methods around policy and query existance
type Library interface {
	// QueryExists checks if the given MRN exists
	QueryExists(ctx context.Context, mrn string) (bool, error)
	// PolicyExists checks if the given MRN exists
	PolicyExists(ctx context.Context, mrn string) (bool, error)
}
