// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/recording"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
	"go.mondoo.com/mql/providers-sdk/v1/upstream"
)

func TestFilterPreprocess(t *testing.T) {
	// given
	filters := []string{
		"namespace1/policy1",
		"namespace2/policy2",
		"//registry.mondoo.com/namespace/namespace3/policies/policy3",
	}

	// when
	preprocessed := preprocessPolicyFilters(filters)

	// then
	assert.Equal(t, []string{
		"//registry.mondoo.com/namespace/namespace1/policies/policy1",
		"//registry.mondoo.com/namespace/namespace2/policies/policy2",
		"//registry.mondoo.com/namespace/namespace3/policies/policy3",
	}, preprocessed)
}

func TestGetUpstreamConfig(t *testing.T) {
	t.Run("with job creds", func(t *testing.T) {
		opts := []ScannerOption{
			AllowJobCredentials(),
		}

		pk, err := os.ReadFile("../testdata/private-key.p8")
		require.NoError(t, err)

		cert, err := os.ReadFile("../testdata/cert.pem")
		require.NoError(t, err)

		job := &Job{
			Inventory: &inventory.Inventory{
				Spec: &inventory.InventorySpec{
					UpstreamCredentials: &upstream.ServiceAccountCredentials{
						ApiEndpoint: "api",
						ScopeMrn:    "space-mrn",
						PrivateKey:  string(pk),
						Certificate: string(cert),
					},
				},
			},
		}
		scanner := NewLocalScanner(opts...)
		_, err = scanner.getUpstreamConfig(false, job)
		require.NoError(t, err)

		_, err = scanner.getUpstreamConfig(true, &Job{})
		require.NoError(t, err)
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Run("without opts", func(t *testing.T) {
		scanner := NewLocalScanner()
		require.NotNil(t, scanner)

		require.Equal(t, recording.Null{}, scanner.recording)
	})
}

func TestWithServerFeatures(t *testing.T) {
	t.Run("activates known server features on the context", func(t *testing.T) {
		ctx := mql.SetFeatures(context.Background(), mql.DefaultFeatures)
		ctx = withServerFeatures(ctx, []string{"TerraformResolveVars"})
		require.True(t, mql.GetFeatures(ctx).IsActive(mql.TerraformResolveVars))
	})

	t.Run("skips unknown features but keeps known ones", func(t *testing.T) {
		ctx := mql.SetFeatures(context.Background(), mql.DefaultFeatures)
		ctx = withServerFeatures(ctx, []string{"NotARealFeature", "TerraformResolveVars"})
		require.True(t, mql.GetFeatures(ctx).IsActive(mql.TerraformResolveVars))
	})

	t.Run("empty list returns the context unchanged", func(t *testing.T) {
		base := mql.SetFeatures(context.Background(), mql.DefaultFeatures)
		require.Equal(t, base, withServerFeatures(base, nil))
	})
}

// TestResolveServerScanParameters_NoUpstream verifies the pre-discovery hook is
// a no-op (context unchanged, nil services) for incognito / credential-less
// scans, so it can be safely called before asset discovery in every scan.
func TestResolveServerScanParameters_NoUpstream(t *testing.T) {
	s := NewLocalScanner()
	base := mql.SetFeatures(context.Background(), mql.DefaultFeatures)

	t.Run("nil upstream", func(t *testing.T) {
		ctx, services, spaceMrn, params, err := s.resolveServerScanParameters(base, nil)
		require.NoError(t, err)
		require.Equal(t, base, ctx)
		require.Nil(t, services)
		require.Empty(t, spaceMrn)
		require.Nil(t, params)
	})

	t.Run("incognito upstream", func(t *testing.T) {
		ctx, services, spaceMrn, params, err := s.resolveServerScanParameters(base, &upstream.UpstreamConfig{
			ApiEndpoint: "https://example.com",
			Incognito:   true,
		})
		require.NoError(t, err)
		require.Equal(t, base, ctx)
		require.Nil(t, services)
		require.Empty(t, spaceMrn)
		require.Nil(t, params)
	})
}

type LocalScannerSuite struct {
	suite.Suite
	ctx  context.Context
	conf mqlc.CompilerConfig
	job  *Job
}

func (s *LocalScannerSuite) SetupSuite() {
	s.ctx = context.Background()
	// @afiune by default, testutils.Local() returns a runtime with auto-update disabled we
	// need to update this function to accept a runtime, for now, patch it after initialization
	runtime := testutils.Local()
	providersRuntime := providers.DefaultRuntime()
	providersRuntime.AutoUpdate = providers.UpdateProvidersConfig{
		Enabled:         true,
		RefreshInterval: 60 * 60,
	}
	s.conf = mqlc.NewConfig(runtime.Schema(), mql.DefaultFeatures)
}

func (s *LocalScannerSuite) BeforeTest(suiteName, testName string) {
	s.job = &Job{
		Inventory: &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{
					{
						Connections: []*inventory.Config{
							{
								Type: "k8s",
								Options: map[string]string{
									"path": "./testdata/1pod.yaml",
								},
								Discover: &inventory.Discovery{
									Targets: []string{"pods"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *LocalScannerSuite) TestRunIncognito_SharedQuery() {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/shared-query.mql.yaml")
	s.Require().NoError(err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: s.conf,
		RemoveFailing:  true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle
	bundleMap := bundle.ToMap()

	ctx := context.Background()
	scanner := NewLocalScanner(DisableProgressBar())
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for k, r := range full.Reports {
		// Verify the score is 100
		s.Equal(uint32(100), r.GetScore().Value)

		p := full.ResolvedPolicies[k]

		// Get the code id for all the executed queries
		executedQueries := []string{}
		for qCodeId := range p.ExecutionJob.Queries {
			executedQueries = append(executedQueries, qCodeId)
		}

		expectedQueries := []string{
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/sshd-01"].CodeId,
		}
		s.ElementsMatch(expectedQueries, executedQueries)
	}
}

func (s *LocalScannerSuite) TestRunIncognito_ExceptionGroups() {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/exception-groups.mql.yaml")
	s.Require().NoError(err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: s.conf,
		RemoveFailing:  true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle
	s.job.PolicyFilters = []string{"asset-policy"}

	ctx := context.Background()
	scanner := NewLocalScanner(DisableProgressBar())
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for k, r := range full.Reports {
		// Verify the score is 100
		s.Equal(uint32(100), r.GetScore().Value)

		p := full.ResolvedPolicies[k]

		queryIdToReportingJob := map[string]*policy.ReportingJob{}
		for _, rj := range p.CollectorJob.ReportingJobs {
			_, ok := queryIdToReportingJob[rj.QrId]
			s.Require().False(ok)
			queryIdToReportingJob[rj.QrId] = rj
		}

		// Make sure the ignored query is ignored
		{
			queryRj := queryIdToReportingJob["//local.cnspec.io/run/local-execution/queries/ignored-query"]
			s.Require().NotNil(queryRj)

			parent := queryRj.Notify[0]
			parentJob := p.CollectorJob.ReportingJobs[parent]
			s.Require().NotNil(parentJob)
			s.Equal(policy.ScoringSystem_IGNORE_SCORE, parentJob.ChildJobs[queryRj.Uuid].Scoring)
		}
		// Make sure the ignored query is reported as disabled
		{
			queryRj := queryIdToReportingJob["//local.cnspec.io/run/local-execution/queries/deactivate-query"]
			s.Require().NotNil(queryRj)
			var child string
			for c := range queryRj.ChildJobs {
				child = c
				break
			}
			s.Equal(policy.ScoringSystem_DISABLED, queryRj.ChildJobs[child].Scoring)
		}
	}
}

func (s *LocalScannerSuite) TestRunIncognito_ExceptionGroups_RejectedReview() {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/exception-groups.mql.yaml")
	s.Require().NoError(err)

	bundle.Policies[1].Groups[1].ReviewStatus = policy.ReviewStatus_REJECTED
	bundle.Policies[1].Groups[2].ReviewStatus = policy.ReviewStatus_REJECTED

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: s.conf,
		RemoveFailing:  true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle
	s.job.PolicyFilters = []string{"asset-policy"}
	bundleMap := bundle.ToMap()

	ctx := context.Background()
	scanner := NewLocalScanner(DisableProgressBar())
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for k, r := range full.Reports {
		// Verify the score is 33
		s.Equal(uint32(33), r.GetScore().Value)

		p := full.ResolvedPolicies[k]

		// Get the code id for all the executed queries
		executedQueries := []string{}
		for qCodeId := range p.ExecutionJob.Queries {
			executedQueries = append(executedQueries, qCodeId)
		}

		expectedQueries := []string{
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId,
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/deactivate-query"].CodeId,
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/sshd-score-01"].CodeId,
		}
		s.ElementsMatch(expectedQueries, executedQueries)

		queryIdToReportingJob := map[string]*policy.ReportingJob{}
		for _, rj := range p.CollectorJob.ReportingJobs {
			_, ok := queryIdToReportingJob[rj.QrId]
			s.Require().False(ok)
			queryIdToReportingJob[rj.QrId] = rj
		}

		// Make sure the ignored query is ignored
		queryRj := queryIdToReportingJob["//local.cnspec.io/run/local-execution/queries/ignored-query"]
		s.Require().NotNil(queryRj)

		parent := queryRj.Notify[0]
		parentJob := p.CollectorJob.ReportingJobs[parent]
		s.Require().NotNil(parentJob)
		impact, ok := parentJob.ChildJobs[queryRj.Uuid]
		s.Require().True(ok)
		s.Require().Nil(impact)
	}
}

func (s *LocalScannerSuite) TestRunIncognito_QueryExceptions() {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/exceptions.mql.yaml")
	s.Require().NoError(err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: s.conf,
		RemoveFailing:  true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle
	s.job.PolicyFilters = []string{"asset-policy"}

	ctx := context.Background()
	scanner := NewLocalScanner(DisableProgressBar())
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for k, r := range full.Reports {
		// Verify the score is 100
		s.Equal(uint32(100), r.GetScore().Value)

		p := full.ResolvedPolicies[k]

		queryIdToReportingJob := map[string]*policy.ReportingJob{}
		for _, rj := range p.CollectorJob.ReportingJobs {
			_, ok := queryIdToReportingJob[rj.QrId]
			s.Require().False(ok)
			queryIdToReportingJob[rj.QrId] = rj
		}

		// Make sure the ignored query is ignored
		queryRj := queryIdToReportingJob["//local.cnspec.io/run/local-execution/queries/ignored-query"]
		s.Require().NotNil(queryRj)

		parent := queryRj.Notify[0]
		parentJob := p.CollectorJob.ReportingJobs[parent]
		s.Require().NotNil(parentJob)
		s.Equal(policy.ScoringSystem_IGNORE_SCORE, parentJob.ChildJobs[queryRj.Uuid].Scoring)
	}
}

func (s *LocalScannerSuite) TestRunIncognito_QueryExceptions_MultipleGroups() {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/exceptions-multiple-groups.mql.yaml")
	s.Require().NoError(err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: s.conf,
		RemoveFailing:  true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle
	s.job.PolicyFilters = []string{"asset-policy"}

	ctx := context.Background()
	scanner := NewLocalScanner(DisableProgressBar())
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for k, r := range full.Reports {
		// Verify the score is 100
		s.Equal(uint32(100), r.GetScore().Value)

		p := full.ResolvedPolicies[k]

		queryIdToReportingJob := map[string]*policy.ReportingJob{}
		for _, rj := range p.CollectorJob.ReportingJobs {
			_, ok := queryIdToReportingJob[rj.QrId]
			s.Require().False(ok)
			queryIdToReportingJob[rj.QrId] = rj
		}

		// Make sure the ignored query is ignored
		queryRj := queryIdToReportingJob["//local.cnspec.io/run/local-execution/queries/ignored-query"]
		s.Require().NotNil(queryRj)

		{
			parent := queryRj.Notify[0]
			parentJob := p.CollectorJob.ReportingJobs[parent]
			s.Require().NotNil(parentJob)
			s.Equal(policy.ScoringSystem_IGNORE_SCORE, parentJob.ChildJobs[queryRj.Uuid].Scoring)
		}
		// Make sure the ignored query is reported as disabled
		{
			queryRj := queryIdToReportingJob["//local.cnspec.io/run/local-execution/queries/deactivate-query"]
			s.Require().NotNil(queryRj)
			var child string
			for c := range queryRj.ChildJobs {
				child = c
				break
			}
			s.Equal(policy.ScoringSystem_DISABLED, queryRj.ChildJobs[child].Scoring)
		}
	}
}

func (s *LocalScannerSuite) TestRunIncognito_Frameworks() {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/compliance-bundle.mql.yaml")
	s.Require().NoError(err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: s.conf,
		RemoveFailing:  true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle

	ctx := context.Background()
	scanner := NewLocalScanner(DisableProgressBar())
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for _, r := range full.Reports {
		s.Contains(r.Scores, "//local.cnspec.io/run/local-execution/controls/mondoo-test-01")
		s.Contains(r.Scores, "//local.cnspec.io/run/local-execution/controls/mondoo-test-02")
	}
}

func (s *LocalScannerSuite) TestRunIncognito_Frameworks_Exceptions_Deactivate() {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/compliance-bundle.mql.yaml")
	s.Require().NoError(err)

	bundle.Frameworks[0].Groups = append(bundle.Frameworks[0].Groups, &policy.FrameworkGroup{
		Type:     policy.GroupType_DISABLE,
		Controls: []*policy.Control{{Mrn: "//local.cnspec.io/run/local-execution/controls/mondoo-test-01"}},
	})
	bundle.Frameworks[0].Groups = append(bundle.Frameworks[0].Groups, &policy.FrameworkGroup{
		Type:         policy.GroupType_DISABLE,
		ReviewStatus: policy.ReviewStatus_REJECTED,
		Controls:     []*policy.Control{{Mrn: "//local.cnspec.io/run/local-execution/controls/mondoo-test-02"}},
	})

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: s.conf,
		RemoveFailing:  true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle

	ctx := context.Background()
	scanner := NewLocalScanner(DisableProgressBar())
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for _, r := range full.Reports {
		s.NotContains(r.Scores, "//local.cnspec.io/run/local-execution/controls/mondoo-test-01")
		s.Contains(r.Scores, "//local.cnspec.io/run/local-execution/controls/mondoo-test-02")
	}
}

func (s *LocalScannerSuite) TestRunIncognito_Frameworks_Exceptions_OutOfScope() {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/compliance-bundle.mql.yaml")
	s.Require().NoError(err)

	bundle.Frameworks[0].Groups = append(bundle.Frameworks[0].Groups, &policy.FrameworkGroup{
		Type:     policy.GroupType_OUT_OF_SCOPE_GROUP,
		Controls: []*policy.Control{{Mrn: "//local.cnspec.io/run/local-execution/controls/mondoo-test-01"}},
	})
	bundle.Frameworks[0].Groups = append(bundle.Frameworks[0].Groups, &policy.FrameworkGroup{
		Type:         policy.GroupType_OUT_OF_SCOPE_GROUP,
		ReviewStatus: policy.ReviewStatus_REJECTED,
		Controls:     []*policy.Control{{Mrn: "//local.cnspec.io/run/local-execution/controls/mondoo-test-02"}},
	})

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: s.conf,
		RemoveFailing:  true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle

	ctx := context.Background()
	scanner := NewLocalScanner(DisableProgressBar())
	res, err := scanner.RunIncognito(ctx, s.job)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	full := res.GetFull()
	s.Require().NotNil(full)

	s.Equal(1, len(full.Reports))

	for _, r := range full.Reports {
		s.NotContains(r.Scores, "//local.cnspec.io/run/local-execution/controls/mondoo-test-01")
		s.Contains(r.Scores, "//local.cnspec.io/run/local-execution/controls/mondoo-test-02")
	}
}

func (s *LocalScannerSuite) TestRunIncognito_ContextCancellation() {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/shared-query.mql.yaml")
	s.Require().NoError(err)

	_, err = bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: s.conf,
		RemoveFailing:  true,
	})
	s.Require().NoError(err)

	s.job.Bundle = bundle

	// Cancel the context immediately — the scan should exit without hanging
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	scanner := NewLocalScanner(DisableProgressBar())
	_, err = scanner.RunIncognito(ctx, s.job)
	// The scan should return a context error, not hang
	s.Error(err)
}

func TestGetAssetDetectBundle(t *testing.T) {
	bundle := getAssetDetectBundle()
	require.NotNil(t, bundle)
	require.NotEmpty(t, bundle.CodeV2.Id)
}

func TestLocalScannerSuite(t *testing.T) {
	suite.Run(t, new(LocalScannerSuite))
}

func TestStampScanSource(t *testing.T) {
	newJob := func() *Job {
		return &Job{
			Inventory: &inventory.Inventory{
				Spec: &inventory.InventorySpec{
					Assets: []*inventory.Asset{
						{Name: "asset-1"},
						{Name: "asset-2", Labels: map[string]string{"existing": "value"}},
					},
				},
			},
		}
	}

	t.Run("stamps every asset", func(t *testing.T) {
		scanner := NewLocalScanner(WithScanSource(ScanSourceService))
		job := newJob()
		scanner.stampScanSource(job)

		for _, a := range job.Inventory.Spec.Assets {
			assert.Equal(t, ScanSourceService, a.Labels[LabelScanSource], "asset %q", a.Name)
		}
		// pre-existing labels are preserved
		assert.Equal(t, "value", job.Inventory.Spec.Assets[1].Labels["existing"])
	})

	t.Run("no-op without a scan source", func(t *testing.T) {
		scanner := NewLocalScanner()
		job := newJob()
		scanner.stampScanSource(job)

		for _, a := range job.Inventory.Spec.Assets {
			_, ok := a.Labels[LabelScanSource]
			assert.False(t, ok, "asset %q should not be labeled", a.Name)
		}
	})

	t.Run("no-op without an inventory", func(t *testing.T) {
		scanner := NewLocalScanner(WithScanSource(ScanSourceService))
		assert.NotPanics(t, func() { scanner.stampScanSource(&Job{}) })
	})
}

func TestNewLocalScannerWithOptions(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		scanner := NewLocalScanner()
		require.NotNil(t, scanner)

		assert.True(t, scanner.autoUpdate)
		assert.Zero(t, scanner.refreshInterval)

		rt, ok := scanner.runtime.(*providers.Runtime)
		require.True(t, ok)
		assert.True(t, rt.AutoUpdate.Enabled)
		assert.Equal(t, defaultRefreshInterval, rt.AutoUpdate.RefreshInterval)
	})

	t.Run("with auto update disabled", func(t *testing.T) {
		scanner := NewLocalScanner(WithAutoUpdate(false))
		require.NotNil(t, scanner)

		require.NotNil(t, scanner.autoUpdate)
		assert.False(t, scanner.autoUpdate)
		assert.Zero(t, scanner.refreshInterval)

		rt, ok := scanner.runtime.(*providers.Runtime)
		require.True(t, ok)
		assert.False(t, rt.AutoUpdate.Enabled)
		assert.Equal(t, defaultRefreshInterval, rt.AutoUpdate.RefreshInterval)
	})

	t.Run("with custom refresh interval", func(t *testing.T) {
		scanner := NewLocalScanner(WithRefreshInterval(1234))
		require.NotNil(t, scanner)

		assert.True(t, scanner.autoUpdate)
		assert.Equal(t, 1234, scanner.refreshInterval)

		rt, ok := scanner.runtime.(*providers.Runtime)
		require.True(t, ok)
		assert.True(t, rt.AutoUpdate.Enabled)
		assert.Equal(t, 1234, rt.AutoUpdate.RefreshInterval)
	})

	t.Run("with scan source", func(t *testing.T) {
		scanner := NewLocalScanner(WithScanSource(ScanSourceInteractive))
		require.NotNil(t, scanner)
		assert.Equal(t, ScanSourceInteractive, scanner.scanSource)
	})

	t.Run("later scan source overrides earlier default", func(t *testing.T) {
		// RunScan prepends an interactive default; callers like `serve` append
		// their own source, which must win.
		scanner := NewLocalScanner(WithScanSource(ScanSourceInteractive), WithScanSource(ScanSourceService))
		require.NotNil(t, scanner)
		assert.Equal(t, ScanSourceService, scanner.scanSource)
	})

	t.Run("with custom runtime ignores auto-update option", func(t *testing.T) {
		// Create a new runtime instance for this test to ensure isolation.
		customRuntime := &providers.Runtime{
			AutoUpdate: providers.UpdateProvidersConfig{
				RefreshInterval: 9999,
				Enabled:         false,
			},
		}
		scanner := NewLocalScanner(WithRuntime(customRuntime), WithAutoUpdate(true), WithRefreshInterval(123))
		require.NotNil(t, scanner)

		assert.Same(t, customRuntime, scanner.runtime)

		rt, ok := scanner.runtime.(*providers.Runtime)
		require.True(t, ok)
		assert.Equal(t, 9999, rt.AutoUpdate.RefreshInterval)
		assert.False(t, rt.AutoUpdate.Enabled, "should not be modified if a custom runtime is provided")
	})
}

// TestClampToServerMax covers the one rule the server ceiling has to obey: it
// is a maximum, never a minimum, and it outranks whatever the client picked --
// including a parallelism the user asked for explicitly.
func TestClampToServerMax(t *testing.T) {
	tests := []struct {
		name        string
		parallelism int
		serverMax   int
		expected    int
	}{
		{name: "no server limit set", parallelism: 8, serverMax: 0, expected: 8},
		{name: "negative server limit is ignored", parallelism: 8, serverMax: -1, expected: 8},
		{name: "server limit above the client value", parallelism: 4, serverMax: 16, expected: 4},
		{name: "server limit equal to the client value", parallelism: 6, serverMax: 6, expected: 6},
		{name: "server limit cuts the client value down", parallelism: 8, serverMax: 2, expected: 2},
		{name: "server can force sequential", parallelism: 8, serverMax: 1, expected: 1},
		{name: "server never raises a sequential scan", parallelism: 1, serverMax: 16, expected: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, clampToServerMax(tc.parallelism, tc.serverMax))
		})
	}
}

// TestScanParametersMaxParallelismDefaultsToUnset pins the wire behavior the
// clamp depends on: a server that says nothing about parallelism must read back
// as zero, which clampToServerMax treats as "no limit" rather than "sequential".
func TestScanParametersMaxParallelismDefaultsToUnset(t *testing.T) {
	var nilParams *policy.ScanParameters
	assert.Equal(t, int32(0), nilParams.GetMaxParallelism(), "a nil response must not throttle the scan")

	empty := &policy.ScanParameters{EnabledFeatures: []string{"some-feature"}}
	assert.Equal(t, int32(0), empty.GetMaxParallelism())
	assert.Equal(t, 8, clampToServerMax(8, int(empty.GetMaxParallelism())))
}

// TestResolveScanParallelismServerOutranksTheClient pins the ordering the
// server ceiling depends on: it is applied after the client has made its
// choice, so it cuts down an explicit --parallelism rather than losing to it.
func TestResolveScanParallelismServerOutranksTheClient(t *testing.T) {
	t.Run("explicit request with no server limit", func(t *testing.T) {
		assert.Equal(t, 20, resolveScanParallelism(20, nil, 0))
	})

	t.Run("server limit cuts an explicit request down", func(t *testing.T) {
		assert.Equal(t, 2, resolveScanParallelism(20, nil, 2),
			"an explicit --parallelism must not escape the server ceiling")
	})

	t.Run("server limit never raises the resolved value", func(t *testing.T) {
		// No roots means no provider opted in, which resolves to sequential.
		assert.Equal(t, 1, resolveScanParallelism(0, nil, 16))
	})
}
