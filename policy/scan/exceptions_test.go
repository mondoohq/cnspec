// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"path/filepath"
	"time"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/cli/config"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

const (
	localQueries = "//local.cnspec.io/run/local-execution/queries/"
	policyMrn    = "//local.cnspec.io/run/local-execution/policies/config-exceptions"
)

func (s *LocalScannerSuite) configExceptionsBundle() (*policy.Bundle, *policy.PolicyBundleMap) {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("./testdata/config-exceptions.mql.yaml")
	s.Require().NoError(err)
	bundleMap, err := bundle.CompileExt(context.Background(), policy.BundleCompileConf{
		CompilerConfig: s.conf,
		RemoveFailing:  true,
	})
	s.Require().NoError(err)
	return bundle, bundleMap
}

// scoreTypes maps each check's MRN to how it scored.
func scoreTypes(report *policy.Report) map[string]uint32 {
	res := map[string]uint32{}
	for mrn, score := range report.Scores {
		res[mrn] = score.Type
	}
	return res
}

// ignored reports whether a check's result is kept out of its parent's score,
// which is where an IGNORED exception takes effect: the check itself still runs
// and has a result.
func ignored(rp *policy.ResolvedPolicy, checkMrn string) bool {
	for _, rj := range rp.CollectorJob.ReportingJobs {
		if rj.QrId != checkMrn {
			continue
		}
		for _, parent := range rj.Notify {
			if impact := rp.CollectorJob.ReportingJobs[parent].ChildJobs[rj.Uuid]; impact != nil {
				return impact.Scoring == policy.ScoringSystem_IGNORE_SCORE
			}
		}
	}
	return false
}

func outcomes(d *policy.ExceptionDecisions) map[string]policy.ExceptionOutcome {
	res := map[string]policy.ExceptionOutcome{}
	for _, item := range d.GetItems() {
		check := item.Entry.CheckUid
		if check == "" {
			check = item.Entry.CheckMrn
		}
		res[check] = item.Outcome
	}
	return res
}

func (s *LocalScannerSuite) TestRunIncognito_UserConfigExceptions() {
	bundle, bundleMap := s.configExceptionsBundle()
	s.job.Bundle = bundle

	past := time.Now().Add(-48 * time.Hour).Format(time.DateOnly)
	user := []config.Exception{
		{Checks: []string{"accepted-check", "no-such-check"}, Action: "risk-accepted", Justification: "accepted"},
		{Checks: []string{"disabled-check"}, Action: "disable", Justification: "not relevant"},
		{Checks: []string{"expired-check"}, Action: "false-positive", Justification: "was wrong", ValidUntil: past},
	}

	scanner := NewLocalScanner(DisableProgressBar(), WithExceptions(user, "~/.config/mondoo/mondoo.yml", false))
	res, err := scanner.RunIncognito(context.Background(), s.job)
	s.Require().NoError(err)
	full := res.GetFull()
	s.Require().NotNil(full)
	s.Require().Len(full.Reports, 1)

	for assetMrn, report := range full.Reports {
		s.Equal(map[string]policy.ExceptionOutcome{
			"accepted-check": policy.ExceptionOutcome_EXCEPTION_OUTCOME_APPLIED,
			"no-such-check":  policy.ExceptionOutcome_EXCEPTION_OUTCOME_UNKNOWN_CHECK,
			"disabled-check": policy.ExceptionOutcome_EXCEPTION_OUTCOME_APPLIED,
			"expired-check":  policy.ExceptionOutcome_EXCEPTION_OUTCOME_EXPIRED,
		}, outcomes(full.ExceptionDecisions[assetMrn]))

		rp := full.ResolvedPolicies[assetMrn]
		scores := scoreTypes(report)
		s.True(ignored(rp, localQueries+"accepted-check"), "a risk-accepted check runs and does not score")
		s.Equal(policy.ScoreType_Disabled, scores[localQueries+"disabled-check"])
		s.False(ignored(rp, localQueries+"expired-check"), "an expired exception does not apply")
		s.Equal(uint32(0), report.Scores[localQueries+"expired-check"].Value)
		// passing (100) and expired (0) score; the accepted and disabled checks do not
		s.Equal(uint32(50), report.Scores[policyMrn].GetValue())

		// A disabled check does not run: its code is not in the execution job.
		s.NotContains(rp.ExecutionJob.Queries, bundleMap.Queries[localQueries+"disabled-check"].CodeId)
		s.Contains(rp.ExecutionJob.Queries, bundleMap.Queries[localQueries+"accepted-check"].CodeId)
	}
}

func (s *LocalScannerSuite) TestRunIncognito_ContextConfigExceptions() {
	bundle, _ := s.configExceptionsBundle()
	s.job.Bundle = bundle

	dir, err := filepath.Abs("./testdata/terraform-exceptions")
	s.Require().NoError(err)

	// The provider attaches the config it found at the scanned root; here the
	// asset already carries it, as a discovering provider would have set it.
	content := `
private_key: should be ignored
exceptions:
  - checks: [accepted-check]
    action: risk-accepted
    justification: staging serves public fixtures
    paths: [infra/staging]
  - checks: [disabled-check]
    action: disable
    justification: only prod
    paths: [infra/prod]
  - checks: [expired-check, passing-check]
    action: workaround
    justification: applies everywhere
  - checks: [disabled-check]
    action: workaround
    justification: no valid action conflict, the prod entry is more specific
    paths: [infra/prod/eu]
`
	s.job.Inventory.Spec.Assets = []*inventory.Asset{{
		Connections: []*inventory.Config{{
			Type:    "terraform-hcl",
			Options: map[string]string{"path": dir},
		}},
		ContextConfig: &inventory.ContextConfig{
			Content:   []byte(content),
			Origin:    &inventory.ConfigOrigin{Provider: "github", Repository: "github.com/acme/infra", Ref: "main", Path: "mondoo.yml"},
			AssetPath: "infra/staging/web",
		},
	}}

	scanner := NewLocalScanner(DisableProgressBar())
	res, err := scanner.RunIncognito(context.Background(), s.job)
	s.Require().NoError(err)
	full := res.GetFull()
	s.Require().NotNil(full)
	s.Require().Len(full.Reports, 1, "errors: %v", full.Errors)

	for assetMrn, report := range full.Reports {
		decisions := full.ExceptionDecisions[assetMrn]
		s.Equal(map[string]policy.ExceptionOutcome{
			"accepted-check": policy.ExceptionOutcome_EXCEPTION_OUTCOME_APPLIED,
			"expired-check":  policy.ExceptionOutcome_EXCEPTION_OUTCOME_APPLIED,
			"passing-check":  policy.ExceptionOutcome_EXCEPTION_OUTCOME_APPLIED,
		}, outcomes(decisions), "entries scoped to infra/prod do not govern infra/staging/web")

		for _, d := range decisions.Items {
			s.Equal("github.com/acme/infra", d.Entry.Source.Repository)
			s.Equal("context", d.Entry.Source.Scope)
			if d.Entry.CheckUid == "accepted-check" {
				s.Equal("infra/staging", d.Entry.MatchedPath)
			}
		}

		rp := full.ResolvedPolicies[assetMrn]
		s.True(ignored(rp, localQueries+"accepted-check"))
		s.True(ignored(rp, localQueries+"expired-check"))
		s.True(ignored(rp, localQueries+"passing-check"))
		s.False(ignored(rp, localQueries+"disabled-check"))
		s.Equal(policy.ScoreType_Result, scoreTypes(report)[localQueries+"disabled-check"])
		// only the failing disabled-check is left to score
		s.Equal(uint32(0), report.Scores[policyMrn].GetValue())
	}
}
