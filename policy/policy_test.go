// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/mrn"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
	"go.mondoo.com/cnspec/v11/policy"
)

var conf mqlc.CompilerConfig

func init() {
	runtime := testutils.Local()
	schema := runtime.Schema()
	conf = mqlc.NewConfig(schema, cnquery.DefaultFeatures)
}

func getChecksums(p *policy.Policy) map[string]string {
	return map[string]string{
		"local content":   p.LocalContentChecksum,
		"local execution": p.LocalExecutionChecksum,
		"graph content":   p.GraphContentChecksum,
		"graph execution": p.GraphExecutionChecksum,
	}
}

func testChecksums(t *testing.T, equality []bool, expected map[string]string, actual map[string]string) {
	keys := []string{"local content", "local execution", "graph content", "graph execution"}
	for i, s := range keys {
		if equality[i] {
			assert.Equal(t, expected[s], actual[s], s+" should be equal")
		} else {
			assert.NotEqual(t, expected[s], actual[s], s+" should not be equal")
		}
	}
}

func makeYamlCategory(d string) []byte {
	return []byte(`
policies:
- uid: test-policy
  groups:
  - title: First group
    ` + d + "\n")
}

func TestPolicyGroupCategory(t *testing.T) {
	tests := []struct {
		title string
		yaml  string
		typ   policy.GroupType
	}{
		{"empty", "", policy.GroupType_UNCATEGORIZED},
		{"uncategorized", "type: uncategorized", policy.GroupType_UNCATEGORIZED},
		{"uncategorized", "type: chapter", policy.GroupType_CHAPTER},
		{"uncategorized", "type: import", policy.GroupType_IMPORT},
		{"uncategorized", "type: override", policy.GroupType_OVERRIDE},
		{"uncategorized", "type: 1", policy.GroupType_CHAPTER},
		{"uncategorized", "type: 2", policy.GroupType_IMPORT},
		{"uncategorized", "type: 3", policy.GroupType_OVERRIDE},
	}
	for i := range tests {
		cur := tests[i]
		t.Run(cur.title, func(t *testing.T) {
			fmt.Println(string(makeYamlCategory(cur.yaml)))
			b, err := policy.BundleFromYAML(makeYamlCategory(cur.yaml))
			require.NoError(t, err)
			require.NotNil(t, b)
			assert.Equal(t, cur.typ, b.Policies[0].Groups[0].Type)
		})
	}
}

func TestUpdateChecksums_Group_ReviewStatus(t *testing.T) {
	loader := policy.DefaultBundleLoader()
	bundle, err := loader.BundleFromPaths("../examples/example.mql.yaml")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = bundle.Compile(ctx, conf.Schema, nil)
	require.NoError(t, err)

	p := bundle.Policies[0]
	err = p.UpdateChecksums(ctx, nil, nil, bundle.ToMap(), conf)
	require.NoError(t, err)

	oldChecksum := p.GraphExecutionChecksum

	p.Groups[0].ReviewStatus = policy.ReviewStatus_REJECTED
	p.InvalidateExecutionChecksums()

	err = p.UpdateChecksums(ctx, nil, nil, bundle.ToMap(), conf)
	require.NoError(t, err)

	// Make sure the execution checksum changes when the review status changed.
	assert.NotEqual(t, oldChecksum, p.GraphExecutionChecksum)
}

func TestPolicyChecksums(t *testing.T) {
	files := []string{
		"../examples/example.mql.yaml",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			loader := policy.DefaultBundleLoader()
			b, err := loader.BundleFromPaths(file)
			require.NoError(t, err)

			// check that the checksum is identical
			ctx := context.Background()

			p := b.Policies[0]
			_, err = b.Compile(ctx, conf.Schema, nil)
			require.NoError(t, err)

			// regular checksum tests

			err = p.UpdateChecksums(ctx, nil, nil, b.ToMap(), conf)
			require.NoError(t, err, "computing initial checksums works")

			checksums := getChecksums(p)
			for k, sum := range checksums {
				assert.NotEmpty(t, sum, k+" checksum should not be empty")
			}

			p.InvalidateLocalChecksums()
			err = p.UpdateChecksums(ctx, nil, nil, b.ToMap(), conf)
			assert.NoError(t, err, "computing checksums again")
			assert.Equal(t, checksums, getChecksums(p), "recomputing yields same checksums")

			// content updates

			contentTests := map[string]func(p *policy.Policy){
				"author changed": func(p *policy.Policy) {
					p.Authors = []*explorer.Author{{Name: "Bob"}}
				},
				"tags changed": func(p *policy.Policy) {
					p.Tags = map[string]string{"key": "val"}
				},
				"name changed": func(p *policy.Policy) {
					p.Name = "nu name"
				},
				"version changed": func(p *policy.Policy) {
					p.Version = "1.2.3"
				},
				"group date changed": func(p *policy.Policy) {
					p.Groups[0].Created = 12345
				},
			}

			runContentTest := func(p *policy.Policy, msg string, f func(p *policy.Policy)) {
				t.Run("content changed: "+msg, func(t *testing.T) {
					checksums = getChecksums(p)
					f(p)
					p.InvalidateLocalChecksums()
					err = p.UpdateChecksums(ctx, nil, nil, b.ToMap(), conf)
					assert.NoError(t, err, "computing checksums")
					testChecksums(t, []bool{false, true, false, true}, checksums, getChecksums(p))
				})
			}

			for k, f := range contentTests {
				runContentTest(p, k, f)
			}

			// special handling for asset policies

			assetMrn, err := mrn.NewMRN("//some.domain/" + policy.MRN_RESOURCE_ASSET + "/assetname123")
			require.NoError(t, err)

			assetPolicy := &policy.Policy{
				Mrn:  assetMrn.String(),
				Name: assetMrn.String(),
			}
			assetBundle := &policy.Bundle{Policies: []*policy.Policy{assetPolicy}}
			_, err = assetBundle.Compile(ctx, conf.Schema, nil)
			require.NoError(t, err)
			err = assetPolicy.UpdateChecksums(ctx, nil, nil, assetBundle.ToMap(), conf)
			require.NoError(t, err)

			runContentTest(assetPolicy, "changing asset policy mrn", func(p *policy.Policy) {
				p.Mrn += "bling"
			})

			// execution updates

			executionTests := map[string]func(){
				"query spec set": func() {
					p.Groups[0].Checks[1] = &explorer.Mquery{
						Mrn: "//local.cnspec.io/run/local-execution/queries/sshd-01",
						Impact: &explorer.Impact{
							Scoring: explorer.ScoringSystem_WORST,
						},
					}
				},
				"query changed": func() {
					// Note: changing the Checksum of a base query doesn't do anything.
					// Only the content matters. Changing the base's CodeIDs/MQL/Type is only
					// effective if the query is taking the mql bits from its base.
					b.Queries[0].CodeId = "12345"
				},
				"query prop changed": func() {
					b.Queries[0].Props = []*explorer.Property{
						{
							Mql:      "1 == 1",
							Checksum: "1234",
						},
					}
				},
				"mrn changed": func() {
					p.Mrn = "normal mrn"
				},
			}

			for k, f := range executionTests {
				t.Run("execution context changed: "+k, func(t *testing.T) {
					checksums = getChecksums(p)
					f()
					p.InvalidateLocalChecksums()
					err = p.UpdateChecksums(ctx, nil, nil, b.ToMap(), conf)
					assert.NoError(t, err, "computing checksums")
					updated := getChecksums(p)
					testChecksums(t, []bool{false, false, false, false}, checksums, updated)
				})
			}
		})
	}
}

func TestPolicyChecksummingWithVariantQueries(t *testing.T) {
	bundleInitial, err := policy.BundleFromYAML([]byte(`
policies:
  - uid: variants-test
    name: Another policy
    version: "1.0.0"
    groups:
      - type: chapter
        queries:
          - uid: testqueryvariants

queries:
  - uid: testqueryvariants
    title: testqueryvariants
    variants:
      - uid: variant1
      - uid: variant2
  - uid: variant1
    mql: 1 == 1
    filters: asset.family.contains("unix")
  - uid: variant2
    mql: 1 == 2
    filters: asset.family.contains("windows")
`))
	require.NoError(t, err)
	pInitial := bundleInitial.Policies[0]
	pInitial.InvalidateLocalChecksums()
	initialBundleMap, err := bundleInitial.Compile(context.Background(), conf.Schema, nil)
	require.NoError(t, err)
	err = pInitial.UpdateChecksums(context.Background(), nil, explorer.QueryMap(initialBundleMap.Queries).GetQuery, initialBundleMap, conf)
	assert.NoError(t, err, "computing checksums")

	bundleUpdated, err := policy.BundleFromYAML([]byte(`
policies:
  - uid: variants-test
    name: Another policy
    version: "1.0.0"
    groups:
      - type: chapter
        queries:
          - uid: testqueryvariants

queries:
  - uid: testqueryvariants
    title: testqueryvariants
    variants:
      - uid: variant1
      - uid: variant2
  - uid: variant1
    mql: 1 == 3
    filters: asset.family.contains("unix")
  - uid: variant2
    mql: 1 == 2
    filters: asset.family.contains("windows")
`))
	require.NoError(t, err)
	pUpdated := bundleUpdated.Policies[0]
	pUpdated.InvalidateLocalChecksums()
	updatedBundleMap, err := bundleUpdated.Compile(context.Background(), conf.Schema, nil)
	require.NoError(t, err)
	err = pUpdated.UpdateChecksums(context.Background(), nil, explorer.QueryMap(updatedBundleMap.Queries).GetQuery, updatedBundleMap, conf)
	assert.NoError(t, err, "computing checksums")

	require.NotEqual(t, pInitial.GraphExecutionChecksum, pUpdated.LocalContentChecksum)
	require.NotEqual(t, pInitial.GraphExecutionChecksum, pUpdated.LocalExecutionChecksum)
	require.NotEqual(t, pInitial.GraphExecutionChecksum, pUpdated.GraphExecutionChecksum)
	require.NotEqual(t, pInitial.GraphContentChecksum, pUpdated.GraphContentChecksum)
}

func TestPolicyChecksummingWithVariantChecks(t *testing.T) {
	bundleInitial, err := policy.BundleFromYAML([]byte(`
policies:
  - uid: variants-test
    name: Another policy
    version: "1.0.0"
    groups:
      - type: chapter
        checks:
          - uid: testqueryvariants

queries:
  - uid: testqueryvariants
    title: testqueryvariants
    variants:
      - uid: variant1
      - uid: variant2
  - uid: variant1
    mql: 1 == 1
    filters: asset.family.contains("unix")
  - uid: variant2
    mql: 1 == 2
    filters: asset.family.contains("windows")
`))
	require.NoError(t, err)
	pInitial := bundleInitial.Policies[0]
	pInitial.InvalidateLocalChecksums()
	initialBundleMap, err := bundleInitial.Compile(context.Background(), conf.Schema, nil)
	require.NoError(t, err)
	err = pInitial.UpdateChecksums(context.Background(), nil, explorer.QueryMap(initialBundleMap.Queries).GetQuery, initialBundleMap, conf)
	assert.NoError(t, err, "computing checksums")

	bundleUpdated, err := policy.BundleFromYAML([]byte(`
policies:
  - uid: variants-test
    name: Another policy
    version: "1.0.0"
    groups:
      - type: chapter
        checks:
          - uid: testqueryvariants

queries:
  - uid: testqueryvariants
    title: testqueryvariants
    variants:
      - uid: variant1
      - uid: variant2
  - uid: variant1
    mql: 1 == 3
    filters: asset.family.contains("unix")
  - uid: variant2
    mql: 1 == 2
    filters: asset.family.contains("windows")
`))
	require.NoError(t, err)
	pUpdated := bundleUpdated.Policies[0]
	pUpdated.InvalidateLocalChecksums()
	updatedBundleMap, err := bundleUpdated.Compile(context.Background(), conf.Schema, nil)
	require.NoError(t, err)
	err = pUpdated.UpdateChecksums(context.Background(), nil, explorer.QueryMap(updatedBundleMap.Queries).GetQuery, updatedBundleMap, conf)
	assert.NoError(t, err, "computing checksums")

	require.NotEqual(t, pInitial.GraphExecutionChecksum, pUpdated.LocalContentChecksum)
	require.NotEqual(t, pInitial.GraphExecutionChecksum, pUpdated.LocalExecutionChecksum)
	require.NotEqual(t, pInitial.GraphExecutionChecksum, pUpdated.GraphExecutionChecksum)
	require.NotEqual(t, pInitial.GraphContentChecksum, pUpdated.GraphContentChecksum)
}

func TestPolicyChecksummingWithVariantChecksWithCycles(t *testing.T) {
	{
		bundleInitial, err := policy.BundleFromYAML([]byte(`
policies:
  - uid: variants-test
    name: Another policy
    version: "1.0.0"
    groups:
      - type: chapter
        checks:
          - uid: testqueryvariants

queries:
  - uid: testqueryvariants
    title: testqueryvariants
    variants:
      - uid: variant1
      - uid: variant2
  - uid: variant1
    variants:
    - uid: variant2
  - uid: variant2
    variants:
    - uid: variant1
    filters: asset.family.contains("windows")
`))
		require.NoError(t, err)
		pInitial := bundleInitial.Policies[0]
		pInitial.InvalidateLocalChecksums()
		_, err = bundleInitial.Compile(context.Background(), conf.Schema, nil)
		require.Contains(t, err.Error(), "variant cycle detected")
	}

	{
		bundleInitial, err := policy.BundleFromYAML([]byte(`
policies:
  - uid: variants-test
    name: Another policy
    version: "1.0.0"
    groups:
      - type: chapter
        queries:
          - uid: testqueryvariants

queries:
  - uid: testqueryvariants
    title: testqueryvariants
    variants:
      - uid: variant1
      - uid: variant2
  - uid: variant1
    variants:
    - uid: variant2
  - uid: variant2
    variants:
    - uid: variant1
    filters: asset.family.contains("windows")
`))
		require.NoError(t, err)
		pInitial := bundleInitial.Policies[0]
		pInitial.InvalidateLocalChecksums()
		_, err = bundleInitial.Compile(context.Background(), conf.Schema, nil)
		require.Contains(t, err.Error(), "variant cycle detected")
	}
}
