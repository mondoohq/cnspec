// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"

	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/explorer/resources"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/types"
)

// DataLake provides additional database calls, that are not accessible to
// external users. We use them with specialized tools only. This limits the
// potential exposure to underlying data and reduces the surface for breaking
// changes.
type DataLake interface {
	// GetPolicyFilters retrieves the list of asset filters for a policy (fast)
	GetPolicyFilters(ctx context.Context, mrn string) ([]*explorer.Mquery, error)

	// QueryExists checks if the given MRN exists
	QueryExists(ctx context.Context, mrn string) (bool, error)
	// PolicyExists checks if the given MRN exists
	PolicyExists(ctx context.Context, mrn string) (bool, error)

	// GetQuery retrieves a given query
	GetQuery(ctx context.Context, mrn string) (*explorer.Mquery, error)
	// SetQuery stores a given query
	// Note: the query must be defined, it cannot be nil
	SetQuery(ctx context.Context, mrn string, query *explorer.Mquery) error

	// GetValidatedPolicy retrieves and if necessary updates the policy
	GetValidatedPolicy(ctx context.Context, mrn string) (*Policy, error)
	// DeletePolicy removes a given policy
	// Note: the MRN has to be valid
	DeletePolicy(ctx context.Context, mrn string) error
	// GetValidatedBundle retrieves and if necessary updates the policy bundle
	// Note: the checksum and graphchecksum of the policy must be computed to the right number
	GetValidatedBundle(ctx context.Context, mrn string) (*Bundle, error)

	// SetFramework stores a given framework in the data lake. Note: it does not
	// store any framework maps, there is a separate call for them.
	SetFramework(ctx context.Context, framework *Framework) error
	// SetFrameworkMaps stores a list of framework maps connecting frameworks
	// to policies.
	SetFrameworkMaps(ctx context.Context, ownerFramework string, maps []*FrameworkMap) error
	// GetFramework retrieves a framework from storage. This does not include
	// framework maps!
	GetFramework(ctx context.Context, mrn string) (*Framework, error)
	// GetFrameworkMaps retrieves a set of framework maps for a given framework
	// from the data lake. This doesn't include controls metadata. If there
	// are no framework maps for this MRN, it returns nil (no error).
	GetFrameworkMaps(ctx context.Context, mrn string) ([]*FrameworkMap, error)
	// MutateAssignments modifies a framework of a given asset.
	MutateAssignments(ctx context.Context, mutation *AssetMutation, createIfMissing bool) error

	// GetRawPolicy retrieves the policy without fixing any invalidations (fast)
	GetRawPolicy(ctx context.Context, mrn string) (*Policy, error)
	// SetPolicy stores a given policy in the data lake
	SetPolicy(ctx context.Context, policy *Policy, filters []*explorer.Mquery) error
	// SetRiskFactor creates and stores a risk factor
	SetRiskFactor(ctx context.Context, riskFactor *RiskFactor) error

	// List all policies for a given owner
	// Note: Owner MRN is required
	ListPolicies(ctx context.Context, ownerMrn string, name string) ([]*Policy, error)

	// DeprecatedV8_MutatePolicy modifies a policy. If it does not find the policy, and if the
	// caller chooses to, it will treat the MRN as an asset and create it + its policy.
	// Deprecated for MutateAssignment.
	DeprecatedV8_MutatePolicy(ctx context.Context, mutation *PolicyMutationDelta, createIfMissing bool) (*Policy, error)
	// SetProps will override properties for a given entity (asset, space, org)
	SetProps(ctx context.Context, req *explorer.PropsReq) error
	// SetAssetResolvedPolicy sets and initialized all fields for an asset's resolved policy
	SetAssetResolvedPolicy(ctx context.Context, assetMrn string, resolvedPolicy *ResolvedPolicy, version ResolvedPolicyVersion) error

	// CachedResolvedPolicy returns the resolved policy if it exists
	CachedResolvedPolicy(ctx context.Context, policyMrn string, assetFilterChecksum string, version ResolvedPolicyVersion) (*ResolvedPolicy, error)
	// GetResolvedPolicy returns the resolved policy for a given asset
	GetResolvedPolicy(ctx context.Context, assetMrn string) (*ResolvedPolicy, error)
	// ResolveQuery looks up a given query and caches it for later access (optional)
	ResolveQuery(ctx context.Context, mrn string) (*explorer.Mquery, error)
	// SetResolvedPolicy to the data store; cached indicates if it was cached from
	// upstream, thus preventing any attempts of resolving it in the client
	SetResolvedPolicy(ctx context.Context, mrn string, resolvedPolicy *ResolvedPolicy, version ResolvedPolicyVersion, cached bool) error

	// GetScore retrieves one score for an asset
	GetScore(ctx context.Context, assetMrn string, scoreID string) (Score, error)
	// GetScoredRisks retrieves risk scores for an asset
	GetScoredRisks(ctx context.Context, assetMrn string) (*ScoredRiskFactors, error)
	// UpdateScores sets the given scores and returns a list of updated IDs
	UpdateScores(ctx context.Context, assetMrn string, scores []*Score) (map[string]struct{}, error)
	// UpdateData sets the list of data value for a given asset and returns a list of updated IDs
	UpdateData(ctx context.Context, assetMrn string, data map[string]*llx.Result) (map[string]types.Type, error)
	// UpdateRisks sets the given risks and returns any that were updated
	UpdateRisks(ctx context.Context, assetMrn string, data []*ScoredRiskFactor) (map[string]struct{}, error)
	// GetResources retrieves previously stored resources about an asset
	GetResources(ctx context.Context, assetMrn string, req []*resources.ResourceDataReq) ([]*llx.ResourceRecording, error)
	// UpdateResources stores resources recording data for a given asset
	UpdateResources(ctx context.Context, assetMrn string, resourcesRecording map[string]*llx.ResourceRecording) error

	// GetReport retrieves all scores and data for a given asset
	GetReport(ctx context.Context, assetMrn string, qrID string) (*Report, error)

	// EnsureAsset makes sure an asset with mrn exists
	EnsureAsset(ctx context.Context, mrn string) error
}
