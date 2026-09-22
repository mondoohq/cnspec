// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
)

// An overlay is a policy that imports another policy and adjusts one of its
// checks. The interesting case is the one the platform actually serves: the
// imported policy is owned by a different scope (Mondoo's) and is not part of
// the uploaded bundle at all. See mondoohq/server#20175.
const overlayBasePolicy = `
owner_mrn: //policy.api.mondoo.app
policies:
- uid: mondoo-aws-security
  name: Mondoo AWS Security
  version: 1.0.0
  groups:
  - title: kms
    filters: "true"
    checks:
    - uid: kms-key-no-public-access
      title: KMS keys must not be publicly accessible
      mql: "false"
`

func overlayServices(t *testing.T) *policy.LocalServices {
	runtime := testutils.LinuxMock()
	_, srv, err := inmemory.NewServices(runtime)
	require.NoError(t, err)
	return srv
}

// baseCheckMrn is the MRN the imported policy owns. An override has to end up
// pointing at exactly this, or it adjusts nothing.
const baseCheckMrn = "//policy.api.mondoo.app/queries/kms-key-no-public-access"

func TestOverlay_CrossScopeMqlOverride(t *testing.T) {
	ctx := context.Background()
	srv := overlayServices(t)

	_, err := srv.SetBundle(ctx, parseBundle(t, overlayBasePolicy))
	require.NoError(t, err)

	// The override names the check by uid, which is the only form the platform
	// accepts - a global query mrn written by hand is rejected by the server's
	// authz layer, and rightly so.
	_, err = srv.SetBundle(ctx, parseBundle(t, `
owner_mrn: //captain.api.mondoo.app/spaces/acme
policies:
- uid: overlay-test
  name: Overlay test
  version: 1.0.0
  groups:
  - type: import
    policies:
    - mrn: //policy.api.mondoo.app/policies/mondoo-aws-security
  - type: override
    title: corrected checks
    filters: "true"
    checks:
    - uid: kms-key-no-public-access
      impact: 10
      action: modify
      mql: "true"
`))
	require.NoError(t, err)

	// The override must NOT have been written to the row the imported policy
	// owns. That row is shared by every policy - and every tenant - using it.
	base, err := srv.DataLake.GetQuery(ctx, baseCheckMrn)
	require.NoError(t, err)
	assert.Equal(t, "false", base.Mql, "the imported policy's own check was rewritten")
	assert.Equal(t, "KMS keys must not be publicly accessible", base.Title)
	assert.Nil(t, base.Impact.GetValue(), "the imported policy's own impact was rewritten")

	// Nor may it have created a query of its own in the uploader's scope: that
	// orphan is what used to make the overlay resolve to nothing.
	_, err = srv.DataLake.GetQuery(ctx, "//captain.api.mondoo.app/spaces/acme/queries/kms-key-no-public-access")
	assert.Error(t, err, "the override was published as a new query in the uploader's scope")

	overlayMrn := "//captain.api.mondoo.app/spaces/acme/policies/overlay-test"
	_, err = srv.Assign(ctx, &policy.PolicyAssignment{
		AssetMrn:   "asset1",
		PolicyMrns: []string{overlayMrn},
	})
	require.NoError(t, err)

	rp, err := srv.Resolve(ctx, &policy.ResolveReq{
		PolicyMrn:    "asset1",
		AssetFilters: []*policy.Mquery{{Mql: "true"}},
	})
	require.NoError(t, err)

	// The overlay resolves to the imported policy's check, running the
	// override's MQL, under the imported check's MRN.
	require.Len(t, rp.ExecutionJob.Queries, 1)
	for _, q := range rp.ExecutionJob.Queries {
		assert.Equal(t, "true", q.Query, "the override's MQL did not reach the execution job")
	}

	qrIds := map[string]*policy.ReportingJob{}
	for _, rj := range rp.CollectorJob.ReportingJobs {
		qrIds[rj.QrId] = rj
	}
	require.Contains(t, qrIds, baseCheckMrn, "the imported check is missing from the resolved policy")
	require.Contains(t, qrIds, "//policy.api.mondoo.app/policies/mondoo-aws-security")
	require.Contains(t, qrIds, overlayMrn)
	assert.Equal(t, policy.ReportingJob_CHECK, qrIds[baseCheckMrn].Type)
}

func TestOverlay_CrossScopeAdjustmentOnlyOverride(t *testing.T) {
	ctx := context.Background()
	srv := overlayServices(t)

	_, err := srv.SetBundle(ctx, parseBundle(t, overlayBasePolicy))
	require.NoError(t, err)

	// No mql: the author only wants to change the impact. This used to fail to
	// compile with "query is not implemented", which is what pushed authors
	// into supplying MQL they never wanted.
	_, err = srv.SetBundle(ctx, parseBundle(t, `
owner_mrn: //captain.api.mondoo.app/spaces/acme
policies:
- uid: overlay-impact
  name: Overlay impact only
  version: 1.0.0
  groups:
  - type: import
    policies:
    - mrn: //policy.api.mondoo.app/policies/mondoo-aws-security
  - type: override
    title: lower the impact
    filters: "true"
    checks:
    - uid: kms-key-no-public-access
      action: modify
      impact: 20
`))
	require.NoError(t, err)

	base, err := srv.DataLake.GetQuery(ctx, baseCheckMrn)
	require.NoError(t, err)
	assert.Equal(t, "false", base.Mql)
	assert.Nil(t, base.Impact.GetValue(), "the imported policy's own impact was rewritten")

	overlayMrn := "//captain.api.mondoo.app/spaces/acme/policies/overlay-impact"
	_, err = srv.Assign(ctx, &policy.PolicyAssignment{
		AssetMrn:   "asset-impact",
		PolicyMrns: []string{overlayMrn},
	})
	require.NoError(t, err)

	rp, err := srv.Resolve(ctx, &policy.ResolveReq{
		PolicyMrn:    "asset-impact",
		AssetFilters: []*policy.Mquery{{Mql: "true"}},
	})
	require.NoError(t, err)

	// The base check still runs its own MQL...
	require.Len(t, rp.ExecutionJob.Queries, 1)
	for _, q := range rp.ExecutionJob.Queries {
		assert.Equal(t, "false", q.Query)
	}

	// ...and the impact the override asked for reaches the check's edge.
	var checkJob *policy.ReportingJob
	for _, rj := range rp.CollectorJob.ReportingJobs {
		if rj.QrId == baseCheckMrn {
			checkJob = rj
		}
	}
	require.NotNil(t, checkJob, "the imported check is missing from the resolved policy")
	var found bool
	for _, impact := range checkJob.ChildJobs {
		if impact.GetValue().GetValue() == 20 {
			found = true
		}
	}
	assert.True(t, found, "the override's impact did not reach the check: %v", checkJob.ChildJobs)
}

func TestOverlay_OverrideTargetingUnknownCheckFails(t *testing.T) {
	ctx := context.Background()
	srv := overlayServices(t)

	_, err := srv.SetBundle(ctx, parseBundle(t, overlayBasePolicy))
	require.NoError(t, err)

	// A typo in the check uid. Nothing owns it, so the override can never do
	// anything - saying so is the whole point.
	_, err = srv.SetBundle(ctx, parseBundle(t, `
owner_mrn: //captain.api.mondoo.app/spaces/acme
policies:
- uid: overlay-typo
  name: Overlay with a typo
  version: 1.0.0
  groups:
  - type: import
    policies:
    - mrn: //policy.api.mondoo.app/policies/mondoo-aws-security
  - type: override
    title: corrected checks
    filters: "true"
    checks:
    - uid: kms-key-no-publik-access
      action: modify
      impact: 20
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "override targets check 'kms-key-no-publik-access'")
}

// An overlay imports a policy that is not in its own bundle. The existence
// check for that policy used to leave a nil placeholder in the compiled bundle
// map, which then crashed the dependency sort (and, before that, made
// SetBundleMap reject "cannot set query ... as it is not defined").
func TestOverlay_ImportOutsideBundleDoesNotPoisonBundleMap(t *testing.T) {
	ctx := context.Background()
	srv := overlayServices(t)

	_, err := srv.SetBundle(ctx, parseBundle(t, overlayBasePolicy))
	require.NoError(t, err)

	b := parseBundle(t, `
owner_mrn: //captain.api.mondoo.app/spaces/acme
policies:
- uid: overlay-plain-import
  name: Plain import
  version: 1.0.0
  groups:
  - type: import
    policies:
    - mrn: //policy.api.mondoo.app/policies/mondoo-aws-security
`)

	bundleMap, err := b.Compile(ctx, srv.Runtime.Schema(), srv.DataLake)
	require.NoError(t, err)
	for mrn, p := range bundleMap.Policies {
		assert.NotNil(t, p, "nil policy placeholder left in the bundle map for %s", mrn)
	}
	for mrn, q := range bundleMap.Queries {
		assert.NotNil(t, q, "nil query placeholder left in the bundle map for %s", mrn)
	}

	// The whole point: this used to panic, then error.
	_, err = srv.SetBundle(ctx, b)
	require.NoError(t, err)
}

// The shape the issue actually reported: the imported policy expresses the
// check as a parent with per-platform variants, and the override names the
// variant leaf.
func TestOverlay_CrossScopeVariantOverride(t *testing.T) {
	ctx := context.Background()
	srv := overlayServices(t)

	_, err := srv.SetBundle(ctx, parseBundle(t, `
owner_mrn: //policy.api.mondoo.app
policies:
- uid: mondoo-aws-security
  name: Mondoo AWS Security
  version: 1.0.0
  groups:
  - title: kms
    checks:
    - uid: kms-key-no-public-access
queries:
- uid: kms-key-no-public-access
  title: KMS keys must not be publicly accessible
  variants:
  - uid: kms-key-no-public-access-terraform
  - uid: kms-key-no-public-access-aws
- uid: kms-key-no-public-access-terraform
  title: KMS (terraform)
  filters: asset.platform == "terraform-plan"
  mql: "false"
- uid: kms-key-no-public-access-aws
  title: KMS (aws)
  filters: asset.platform == "aws"
  mql: "false"
`))
	require.NoError(t, err)

	_, err = srv.SetBundle(ctx, parseBundle(t, `
owner_mrn: //captain.api.mondoo.app/spaces/acme
policies:
- uid: overlay-variant
  name: Overlay variant
  version: 1.0.0
  groups:
  - type: import
    policies:
    - mrn: //policy.api.mondoo.app/policies/mondoo-aws-security
  - type: override
    title: corrected checks
    filters: asset.platform == "terraform-plan"
    checks:
    - uid: kms-key-no-public-access-terraform
      impact: 10
      action: modify
      mql: "true"
`))
	require.NoError(t, err)

	// the variant the override replaced must keep its own definition upstream
	variant, err := srv.DataLake.GetQuery(ctx, "//policy.api.mondoo.app/queries/kms-key-no-public-access-terraform")
	require.NoError(t, err)
	assert.Equal(t, "false", variant.Mql, "the imported policy's variant was rewritten")

	overlayMrn := "//captain.api.mondoo.app/spaces/acme/policies/overlay-variant"
	_, err = srv.Assign(ctx, &policy.PolicyAssignment{
		AssetMrn:   "asset-variant",
		PolicyMrns: []string{overlayMrn},
	})
	require.NoError(t, err)

	rp, err := srv.Resolve(ctx, &policy.ResolveReq{
		PolicyMrn:    "asset-variant",
		AssetFilters: []*policy.Mquery{{Mql: `asset.platform == "terraform-plan"`}},
	})
	require.NoError(t, err)

	require.Len(t, rp.ExecutionJob.Queries, 1)
	for _, q := range rp.ExecutionJob.Queries {
		assert.Equal(t, "true", q.Query, "the override's MQL did not replace the variant")
	}

	qrIds := map[string]bool{}
	for _, rj := range rp.CollectorJob.ReportingJobs {
		qrIds[rj.QrId] = true
	}
	// both the parent check and the overridden variant keep their own MRNs
	assert.True(t, qrIds["//policy.api.mondoo.app/queries/kms-key-no-public-access"],
		"the parent check is missing: %v", qrIds)
	assert.True(t, qrIds["//policy.api.mondoo.app/queries/kms-key-no-public-access-terraform"],
		"the overridden variant is missing: %v", qrIds)
}
