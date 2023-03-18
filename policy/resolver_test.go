package policy_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnspec/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/policy"
)

type testAsset struct {
	asset    string
	policies []string
}

func parseBundle(t *testing.T, data string) *policy.Bundle {
	res, err := policy.BundleFromYAML([]byte(data))
	require.NoError(t, err)
	return res
}

func initResolver(t *testing.T, assets []*testAsset, bundles []*policy.Bundle) *policy.LocalServices {
	_, srv, err := inmemory.NewServices(nil)
	require.NoError(t, err)

	for i := range bundles {
		bundle := bundles[i]
		_, err := srv.SetBundle(context.Background(), bundle)
		require.NoError(t, err)
	}

	for i := range assets {
		asset := assets[i]
		_, err := srv.Assign(context.Background(), &policy.PolicyAssignment{
			AssetMrn:   asset.asset,
			PolicyMrns: asset.policies,
		})
		require.NoError(t, err)
	}

	return srv
}

func policyMrn(uid string) string {
	return "//test.sth/policies/" + uid
}

func TestResolve_EmptyPolicy(t *testing.T) {
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
`)

	srv := initResolver(t, []*testAsset{
		{"asset1", []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})

	t.Run("resolve w/o filters", func(t *testing.T) {
		_, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn: policyMrn("policy1"),
		})
		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = asset does not match any of the activated policies")
	})

	t.Run("resolve with empty filters", func(t *testing.T) {
		_, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{}},
		})
		assert.EqualError(t, err, "failed to compile query: failed to compile query '': query is not implemented ''")
	})

	t.Run("resolve with random filters", func(t *testing.T) {
		_, err := srv.Resolve(context.Background(), &policy.ResolveReq{
			PolicyMrn:    policyMrn("policy1"),
			AssetFilters: []*explorer.Mquery{{Mql: "true"}},
		})
		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = asset does not support any policy\nfilter supported by policies:\n\n\nasset supports the following filters:\n")
	})
}

func TestResolve_SimplePolicy(t *testing.T) {
	b := parseBundle(t, `
owner_mrn: //test.sth
policies:
- uid: policy1
  groups:
  - type: chapter
    filters: "true"
    checks:
    - uid: check1
      mql: asset.name == props.name
      props:
      - uid: name
        mql: return "definitely not the asset name"
    queries:
    - uid: query1
      mql: asset{*}
`)

	srv := initResolver(t, []*testAsset{
		{"asset1", []string{policyMrn("policy1")}},
	}, []*policy.Bundle{b})
	rp, err := srv.Resolve(context.Background(), &policy.ResolveReq{
		PolicyMrn:    policyMrn("policy1"),
		AssetFilters: []*explorer.Mquery{{Mql: "true"}},
	})
	require.NoError(t, err)
	require.NotNil(t, rp)
}
