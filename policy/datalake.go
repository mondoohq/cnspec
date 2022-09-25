package policy

import (
	"context"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/types"
)

// DataLake provides additional database calls, that are not accessible to
// external users. We use them with specialized tools only. This limits the
// potential exposure to underlying data and reduces the surface for breaking
// changes.
type DataLake interface {
	// GetPolicyFilters retrieves the list of asset filters for a policy (fast)
	GetPolicyFilters(ctx context.Context, mrn string) ([]*Mquery, error)

	// QueryExists checks if the given MRN exists
	QueryExists(ctx context.Context, mrn string) (bool, error)
	// PolicyExists checks if the given MRN exists
	PolicyExists(ctx context.Context, mrn string) (bool, error)

	// GetQuery retrieves a given query
	GetQuery(ctx context.Context, mrn string) (*Mquery, error)
	// SetQuery stores a given query
	// Note: the query must be defined, it cannot be nil
	SetQuery(ctx context.Context, mrn string, query *Mquery, isScored bool) error

	// GetValidatedPolicy retrieves and if necessary updates the policy
	GetValidatedPolicy(ctx context.Context, mrn string) (*Policy, error)
	// DeletePolicy removes a given policy
	// Note: the MRN has to be valid
	DeletePolicy(ctx context.Context, mrn string) error
	// GetValidatedBundle retrieves and if necessary updates the policy bundle
	// Note: the checksum and graphchecksum of the policy must be computed to the right number
	GetValidatedBundle(ctx context.Context, mrn string) (*PolicyBundle, error)

	// GetRawPolicy retrieves the policy without fixing any invalidations (fast)
	GetRawPolicy(ctx context.Context, mrn string) (*Policy, error)
	// SetPolicy stores a given policy in the data lake
	SetPolicy(ctx context.Context, policy *Policy, filters []*Mquery) error
	// // GetRawBundle retrieves the policy bundle without fixing any invalidations (fast)
	// // Returns true if invalidated
	// GetRawBundle(ctx context.Context, mrn string) (*PolicyBundle, bool, error)

	// List all policies for a given owner
	// Note: Owner MRN is required
	ListPolicies(ctx context.Context, ownerMrn string, name string) ([]*Policy, error)
	// // ListPublic queries all policies that are public
	// ListPublicPolicies(ctx context.Context, ownerMrn string, name string) ([]*Policy, error)

	// MutatePolicy modifies a policy. If it does not find the policy, and if the
	// caller chooses to, it will treat the MRN as an asset and create it + its policy
	MutatePolicy(ctx context.Context, mutation *PolicyMutationDelta, createIfMissing bool) (*Policy, error)
	// // GetAssignedPolicies for a given asset
	// GetAssignedPolicies(ctx context.Context, assetMrn string) ([]*Policy, error)
	// SetAssetResolvedPolicy sets and initialized all fields for an asset's resolved policy
	SetAssetResolvedPolicy(ctx context.Context, assetMrn string, resolvedPolicy *ResolvedPolicy, version ResolvedPolicyVersion) error

	// CachedResolvedPolicy returns the resolved policy if it exists
	CachedResolvedPolicy(ctx context.Context, policyMrn string, assetFilterChecksum string, version ResolvedPolicyVersion) (*ResolvedPolicy, error)
	// ResolveQuery looks up a given query and caches it for later access (optional)
	ResolveQuery(ctx context.Context, mrn string, cache map[string]interface{}) (*Mquery, error)
	// SetResolvedPolicy to the data store; cached indicates if it was cached from
	// upstream, thus preventing any attempts of resolving it in the client
	SetResolvedPolicy(ctx context.Context, mrn string, resolvedPolicy *ResolvedPolicy, version ResolvedPolicyVersion, cached bool) error

	// // GetCollectorJob returns the collector job for a given asset
	// GetCollectorJob(ctx context.Context, assetMrn string) (*CollectorJob, error)
	// // GetResolvedPolicy returns the resolved policy for a given asset
	// GetResolvedPolicy(ctx context.Context, assetMrn string) (*ResolvedPolicy, error)
	// GetScore retrieves one score for an asset
	GetScore(ctx context.Context, assetMrn string, scoreID string) (Score, error)
	// // GetScores retrieves a map of scores for an asset
	// GetScores(ctx context.Context, assetMrn string, scoreIDs []string) (map[string]*Score, error)
	// // GetAssetsScore retrieves the given score for all assets; If the score is empty,
	// // it instead gets the asset's main score
	// GetAssetsScore(ctx context.Context, assets []string, scoreID string) (map[string]*Score, error)
	// // GetSpaceAssetsScore retrieves all scores from all assets within a space
	// GetSpaceAssetsScore(ctx context.Context, space string) (map[string]*Score, int32, error)
	// // DumpAssetScores retrieves a map of requested data fields for an asset
	// DumpAssetScores(ctx context.Context, assetMrn string) (map[string]*Score, error)
	// UpdateScores sets the given scores and returns true if any were updated
	UpdateScores(ctx context.Context, assetMrn string, scores []*Score) (map[string]struct{}, error)
	// // DataExists checks if data was collected for the given query
	// DataExists(ctx context.Context, assetMrn string, checksum string, dataType types.Type) (bool, error)
	// // GetData retrieves a map of requested data fields for an asset
	// GetData(ctx context.Context, assetMrn string, fields map[string]types.Type) (map[string]*llx.Result, error)
	// // DumpAssetData retrieves a map of requested data fields for an asset
	// DumpAssetData(ctx context.Context, assetMrn string) (map[string]*llx.Result, error)
	// UpdateData sets the list of data value for a given asset and returns a list of updated IDs
	UpdateData(ctx context.Context, assetMrn string, data map[string]*llx.Result) (map[string]types.Type, error)

	// GetReport retrieves all scores and data for a given asset
	GetReport(ctx context.Context, assetMrn string, qrID string) (*Report, error)
	// // GetSpaceStatistics returns the latest stats for a space
	// GetSpaceStatistics(ctx context.Context, spaceMrn string) (*SpaceStatistics, error)
	// // SetSpaceStatistics stores the latest stats for a space
	// SetSpaceStatistics(ctx context.Context, spaceMrn string, stats *SpaceStatistics) error
	// // GetVulnerabilityReport returns all advisories and cves found for a space or asset
	// GetVulnerabilityReport(ctx context.Context, req *VulnerabilityReportRequest) (*VulnerabilityReport, error)

	// // GetVulnerableAssets returns a summary of paginated vulnerable assets for a space or a single asset.
	// GetVulnerableAssets(ctx context.Context, req *VulnerableAssetsRequest) (*VulnerableAssetsReport, error)
	// // GetVulnerabilitiesSummary returns a summary of paginated CVEs for a space or for a single asset.
	// GetVulnerabilitiesSummary(ctx context.Context, req *VulnerabilitiesSummaryRequest) (*VulnerabilitiesSummaryReport, error)
	// // GetAdvisoriesSummary returns a summary of paginated advisories for a space or for a single asset.
	// GetAdvisoriesSummary(ctx context.Context, req *AdvisoriesSummaryRequest) (*AdvisoriesSummaryReport, error)
	// // GetVulnerabilitiesByFilter returns a list of all vulnerabilities with the filters applied.
	// GetVulnerabilitiesByFilter(ctx context.Context, req *VulnerabilitiesQuery) (*VulnerabilitiesResponse, error)

	// // returns a policy report summary for an entire space
	// GetSpaceReport(ctx context.Context, in *SpaceReportRequest) (*SpaceReport, error)

	// // returns a report for a specific policy for an entire space
	// GetSpacePolicyReport(ctx context.Context, in *SpacePolicyReportRequest) (*SpacePolicyReport, error)

	// // get a specific cicd project
	// GetCicdProject(ctx context.Context, in *Mrn) (*CicdProject, error)
	// // list cicd projects
	// ListCicdProjects(ctx context.Context, in *ListCicdProjectsRequest) (*CicdProjectsPage, error)
	// // remove a cicd project
	// DeleteCicdProject(ctx context.Context, in *Mrn) (*DeleteCicdProjectConfirmation, error)
	// // remove many cicd projects
	// DeleteCicdProjects(ctx context.Context, in *DeleteCicdProjectsRequest) (*DeleteCicdProjectsConfirmation, error)

	// // ExistAsset checks if an asset already exists
	// ExistAsset(ctx context.Context, mrn string) (bool, error)
	// EnsureAsset makes sure an asset with mrn exists
	EnsureAsset(ctx context.Context, mrn string) error
	// // EnsureAssetScore makes sure an asset score exists
	// EnsureAssetScore(ctx context.Context, asset string, score string) error
	// // Synchronizes an asset with platform identifier into a space
	// SyncAsset(ctx context.Context, entry *Asset, isComplete bool) (*Asset, error)
	// // SetAsset creates or updates an asset
	// SetAsset(ctx context.Context, asset *Asset) (*Asset, error)
	// UpdateAssetRelationships(context.Context, *UpdateAssetRelationshipsRequest) (*UpdateAssetRelationshipsResponse, error)
	// // GetAsset retrieves an asset
	// GetAsset(ctx context.Context, mrn string) (*Asset, error)
	// // DeleteAsset removes an asset
	// DeleteAsset(ctx context.Context, mrn string) error
	// // DeleteAssets removes many assets
	// DeleteAssets(ctx context.Context, in *DeleteAssetsRequest) (*DeleteAssetsConfirmation, error)
	// // ListAssets lists assets based on the filter
	// ListAssets(ctx context.Context, f *AssetSearchFilter) (*AssetsPage, error)
	// // GetAssetCount gives back the count of assets
	// GetAssetCount(ctx context.Context, f *AssetSearchFilter) (*GetAssetCountResponse, error)
	// // RemoveTerminatedAssets removes all the terminated assets in a space
	// RemoveTerminatedAssets(ctx context.Context, in *RemoveTerminatedAssetsRequest) (*RemoveTerminatedAssetsConfirmation, error)
	// // SetAssetAnnotations sets the annotations for an asset
	// SetAssetAnnotations(ctx context.Context, in *SetAssetAnnotationsRequest) (*SetAssetAnnotationsResponse, error)
	// // ListLabelsAndAnnotations sets the annotations for an asset
	// ListLabelsAndAnnotations(ctx context.Context, in *ListLabelsAndAnnotationsRequest) (*ListLabelsAndAnnotationsResponse, error)

	// CalculateAssetGroupStatistics(ctx context.Context, spaceMrn string) ([]*AssetGroupStatistics, error)
	// GetRelatedAssetCounts(ctx context.Context, req *GetRelatedAssetCountsRequest) (*GetRelatedAssetCountsResponse, error)

	// // SetNowProvider sets the function that is used to get the current time
	// // This is for testing only. You'll at best cause a data race if you
	// // call this in non testing situations
	// SetNowProvider(f func() time.Time)
}
