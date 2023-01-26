package policy

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/mrn"
	"sigs.k8s.io/yaml"
)

func getChecksums(p *Policy) map[string]string {
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

func TestPolicyChecksums(t *testing.T) {
	f, err := os.ReadFile("../examples/example.mql.yaml")
	require.NoError(t, err)

	b := Bundle{}
	err = yaml.Unmarshal(f, &b)
	require.NoError(t, err)

	// check that the checksum is identical
	ctx := context.Background()

	p := b.Policies[0]
	_, err = b.Compile(ctx, nil)
	require.NoError(t, err)

	// regular checksum tests

	err = p.UpdateChecksums(ctx, nil, nil, b.ToMap())
	require.NoError(t, err, "computing initial checksums works")

	checksums := getChecksums(p)
	for k, sum := range checksums {
		assert.NotEmpty(t, sum, k+" checksum should not be empty")
	}

	p.InvalidateLocalChecksums()
	err = p.UpdateChecksums(ctx, nil, nil, b.ToMap())
	assert.NoError(t, err, "computing checksums again")
	assert.Equal(t, checksums, getChecksums(p), "recomputing yields same checksums")

	// content updates

	contentTests := map[string]func(p *Policy){
		"author changed": func(p *Policy) {
			p.Authors = []*explorer.Author{{Name: "Bob"}}
		},
		"tags changed": func(p *Policy) {
			p.Tags = map[string]string{"key": "val"}
		},
		"name changed": func(p *Policy) {
			p.Name = "nu name"
		},
		"version changed": func(p *Policy) {
			p.Version = "1.2.3"
		},
		"spec date changed": func(p *Policy) {
			p.Groups[0].Created = 12345
		},
	}

	runContentTest := func(p *Policy, msg string, f func(p *Policy)) {
		t.Run("content changed: "+msg, func(t *testing.T) {
			checksums = getChecksums(p)
			f(p)
			p.InvalidateLocalChecksums()
			err = p.UpdateChecksums(ctx, nil, nil, b.ToMap())
			assert.NoError(t, err, "computing checksums")
			testChecksums(t, []bool{false, true, false, true}, checksums, getChecksums(p))
		})
	}

	for k, f := range contentTests {
		runContentTest(p, k, f)
	}

	// special handling for asset policies

	assetMrn, err := mrn.NewMRN("//some.domain/" + MRN_RESOURCE_ASSET + "/assetname123")
	require.NoError(t, err)

	assetPolicy := &Policy{
		Mrn:  assetMrn.String(),
		Name: assetMrn.String(),
	}
	assetBundle := &Bundle{Policies: []*Policy{assetPolicy}}
	assetBundle.Compile(ctx, nil)
	assetPolicy.UpdateChecksums(ctx, nil, nil, assetBundle.ToMap())

	runContentTest(assetPolicy, "changing asset policy mrn", func(p *Policy) {
		p.Mrn += "bling"
	})

	// execution updates

	executionTests := map[string]func(){
		"query changed": func() {
			b.Queries[0].CodeId = "12345"
		},
		"query spec set": func() {
			p.Groups[0].Checks = []*explorer.Mquery{
				{
					Mrn: "//local.cnspec.io/run/local-execution/queries/sshd-01",
					Impact: &explorer.Impact{
						Scoring: explorer.Impact_WORST,
					},
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
			err = p.UpdateChecksums(ctx, nil, nil, b.ToMap())
			assert.NoError(t, err, "computing checksums")
			testChecksums(t, []bool{false, false, false, false}, checksums, getChecksums(p))
		})
	}
}
