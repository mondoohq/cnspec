// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/mqlc"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/recording"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/testutils"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnspec/v10/policy"
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
						ParentMrn:   "space-mrn",
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

type LocalScannerSuite struct {
	suite.Suite
	ctx  context.Context
	conf mqlc.CompilerConfig
	job  *Job
}

func (s *LocalScannerSuite) SetupSuite() {
	s.ctx = context.Background()
	runtime := testutils.Local()
	s.conf = mqlc.NewConfig(runtime.Schema(), cnquery.DefaultFeatures)
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
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId,
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
		queryRj := queryIdToReportingJob[bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId]
		s.Require().NotNil(queryRj)

		parent := queryRj.Notify[0]
		parentJob := p.CollectorJob.ReportingJobs[parent]
		s.Require().NotNil(parentJob)
		s.Equal(explorer.ScoringSystem_IGNORE_SCORE, parentJob.ChildJobs[queryRj.Uuid].Scoring)
	}
}

func (s *LocalScannerSuite) TestRunIncognito_ExceptionGroups_RejectedReview() {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/exception-groups.mql.yaml")
	s.Require().NoError(err)

	bundle.Policies[1].Groups[0].ReviewStatus = policy.ReviewStatus_REJECTED
	bundle.Policies[1].Groups[1].ReviewStatus = policy.ReviewStatus_REJECTED

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
		// Verify the score is 16
		s.Equal(uint32(16), r.GetScore().Value)

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
		queryRj := queryIdToReportingJob[bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId]
		s.Require().NotNil(queryRj)

		parent := queryRj.Notify[0]
		parentJob := p.CollectorJob.ReportingJobs[parent]
		s.Require().NotNil(parentJob)
		s.Equal(explorer.ScoringSystem_SCORING_UNSPECIFIED, parentJob.ChildJobs[queryRj.Uuid].Scoring)
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
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId,
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
		queryRj := queryIdToReportingJob[bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId]
		s.Require().NotNil(queryRj)

		parent := queryRj.Notify[0]
		parentJob := p.CollectorJob.ReportingJobs[parent]
		s.Require().NotNil(parentJob)
		s.Equal(explorer.ScoringSystem_IGNORE_SCORE, parentJob.ChildJobs[queryRj.Uuid].Scoring)
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
			bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId,
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
		queryRj := queryIdToReportingJob[bundleMap.Queries["//local.cnspec.io/run/local-execution/queries/ignored-query"].CodeId]
		s.Require().NotNil(queryRj)

		parent := queryRj.Notify[0]
		parentJob := p.CollectorJob.ReportingJobs[parent]
		s.Require().NotNil(parentJob)
		s.Equal(explorer.ScoringSystem_IGNORE_SCORE, parentJob.ChildJobs[queryRj.Uuid].Scoring)
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
		Type:     policy.GroupType_OUT_OF_SCOPE,
		Controls: []*policy.Control{{Mrn: "//local.cnspec.io/run/local-execution/controls/mondoo-test-01"}},
	})
	bundle.Frameworks[0].Groups = append(bundle.Frameworks[0].Groups, &policy.FrameworkGroup{
		Type:         policy.GroupType_OUT_OF_SCOPE,
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

func TestLocalScannerSuite(t *testing.T) {
	suite.Run(t, new(LocalScannerSuite))
}
