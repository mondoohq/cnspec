// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"google.golang.org/protobuf/proto"
)

// 15 assets, 0 reports, 15 errors -- every asset failed to scan. A viewer must
// be able to tell this apart from "assets with no findings", so the artifact has
// to carry the errors map through untouched.
const fixtureK8s = "./testdata/report-k8s.json"

func loadFixture(t *testing.T, path string) *policy.ReportCollection {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	res := &policy.ReportCollection{}
	require.NoError(t, json.Unmarshal(raw, res))
	return res
}

// loadUbuntu is the recorded scan of one Ubuntu host: 1 asset, 1 report, 0
// errors -- the single-asset happy path, carrying a bundle, resolved policies,
// raw llx data and a vuln report. It is shared with the other reporters through
// internal/reportfixture rather than read from a path here.
func loadUbuntu(t *testing.T) *policy.ReportCollection {
	t.Helper()
	res, err := reportfixture.UbuntuScan()
	require.NoError(t, err)
	return res
}

func loadK8s(t *testing.T) *policy.ReportCollection {
	t.Helper()
	return loadFixture(t, fixtureK8s)
}

// TestCollectionRoundTrip is the proof that json-full is lossless: load the
// fixture, write it, load it back, and require the two collections to be equal
// as protobuf messages -- not merely similar-looking JSON.
func TestCollectionRoundTrip(t *testing.T) {
	fixtures := map[string]func(*testing.T) *policy.ReportCollection{
		"report-ubuntu": loadUbuntu,
		"report-k8s":    loadK8s,
	}
	for name, load := range fixtures {
		t.Run(name, func(t *testing.T) {
			orig := load(t)

			var first bytes.Buffer
			require.NoError(t, WriteCollection(orig, &first))

			back, err := LoadCollection(bytes.NewReader(first.Bytes()))
			require.NoError(t, err)

			require.True(t, proto.Equal(orig, back),
				"collection changed across a write/load round-trip")

			// writing the reloaded collection must reproduce the same bytes,
			// so the artifact is stable across repeated re-serialization
			var second bytes.Buffer
			require.NoError(t, WriteCollection(back, &second))
			require.Equal(t, first.String(), second.String())

			// and the shape survives a second full round-trip
			again, err := LoadCollection(bytes.NewReader(second.Bytes()))
			require.NoError(t, err)
			require.True(t, proto.Equal(orig, again))
		})
	}
}

// TestCollectionKeepsErroredAssets pins the property the k8s fixture exists
// for. All fifteen assets failed to scan; none of them produced a report.
func TestCollectionKeepsErroredAssets(t *testing.T) {
	orig := loadK8s(t)
	require.Len(t, orig.Assets, 15)
	require.Len(t, orig.Errors, 15)
	require.Empty(t, orig.Reports)

	var buf bytes.Buffer
	require.NoError(t, WriteCollection(orig, &buf))

	back, err := LoadCollection(&buf)
	require.NoError(t, err)

	assert.Len(t, back.Assets, 15)
	assert.Len(t, back.Errors, 15)
	assert.Empty(t, back.Reports)

	// the error strings themselves are what a viewer renders, so compare them
	// verbatim rather than counting them
	for mrn, msg := range orig.Errors {
		require.Contains(t, back.Errors, mrn)
		assert.Equal(t, msg, back.Errors[mrn])
		assert.NotEmpty(t, msg)
	}

	// every errored asset is still identifiable
	for mrn := range orig.Assets {
		require.Contains(t, back.Assets, mrn)
		assert.Equal(t, orig.Assets[mrn].Name, back.Assets[mrn].Name)
	}
}

// TestCollectionKeepsFullFidelity checks the specific things the reduced
// formats drop, since proto.Equal on an accidentally-empty pair would pass.
func TestCollectionKeepsFullFidelity(t *testing.T) {
	orig := loadUbuntu(t)

	var buf bytes.Buffer
	require.NoError(t, WriteCollection(orig, &buf))
	back, err := LoadCollection(&buf)
	require.NoError(t, err)

	require.NotNil(t, back.Bundle, "bundle is dropped by every other format")
	assert.NotEmpty(t, back.Bundle.Policies)
	assert.NotEmpty(t, back.Bundle.Queries)
	assert.NotEmpty(t, back.ResolvedPolicies)
	assert.NotEmpty(t, back.VulnReports)
	require.Len(t, back.Reports, 1)

	// query documentation: titles, docs, remediation, impact
	var withDocs, withRemediation, withImpact int
	for _, q := range back.Bundle.Queries {
		if q.Title != "" {
			withDocs++
		}
		if q.Docs != nil && q.Docs.Remediation != nil {
			withRemediation++
		}
		if q.Impact != nil {
			withImpact++
		}
	}
	assert.NotZero(t, withDocs)
	assert.NotZero(t, withRemediation)
	assert.NotZero(t, withImpact)

	for mrn, report := range back.Reports {
		// (*ReportCollection).ToJSON() nils this out; json-full must not
		assert.NotEmpty(t, report.Data, "report data was dropped for %s", mrn)
		assert.NotEmpty(t, report.Scores)
		assert.Equal(t, len(orig.Reports[mrn].Data), len(report.Data))
		assert.Equal(t, len(orig.Reports[mrn].Scores), len(report.Scores))
	}

	// the resolved policies carry the collector job the traversal walks
	for mrn, rp := range back.ResolvedPolicies {
		require.NotNil(t, rp.CollectorJob, "collector job dropped for %s", mrn)
		assert.NotEmpty(t, rp.CollectorJob.ReportingJobs)
		assert.NotEmpty(t, rp.ExecutionJob.Queries)
	}
}

// TestCollectionRawLlxData proves the base64-encoded llx primitives survive,
// which is the part encoding/json handles differently from protojson.
func TestCollectionRawLlxData(t *testing.T) {
	orig := loadUbuntu(t)

	var buf bytes.Buffer
	require.NoError(t, WriteCollection(orig, &buf))
	back, err := LoadCollection(&buf)
	require.NoError(t, err)

	var compared, nils int
	for mrn, report := range orig.Reports {
		for id, res := range report.Data {
			require.Contains(t, back.Reports[mrn].Data, id)
			got := back.Reports[mrn].Data[id]
			require.True(t, proto.Equal(res, got), "llx result %s changed", id)

			// the fixture contains map entries whose value is literally null;
			// a key that exists with no result is not the same as an absent
			// key, so the artifact has to keep both
			if res == nil {
				assert.Nil(t, got, "null llx result %s came back non-nil", id)
				nils++
				continue
			}
			if len(res.GetData().GetValue()) > 0 {
				assert.Equal(t, res.GetData().GetValue(), got.GetData().GetValue())
				compared++
			}
		}
	}
	assert.NotZero(t, compared, "fixture carried no raw llx values to compare")
	assert.NotZero(t, nils, "fixture carried no null llx results to compare")
}

// TestCollectionDoesNotMutateReceiver guards against the ToJSON() trap: it
// nils Reports[k].Data on its own input.
func TestCollectionDoesNotMutateReceiver(t *testing.T) {
	orig := loadUbuntu(t)
	before := loadUbuntu(t)

	var buf bytes.Buffer
	require.NoError(t, WriteCollection(orig, &buf))

	require.True(t, proto.Equal(before, orig), "WriteCollection mutated its input")
}

func TestWriteCollectionNil(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, WriteCollection(nil, &buf))

	back, err := LoadCollection(&buf)
	require.NoError(t, err)
	assert.Empty(t, back.Assets)
	assert.Empty(t, back.Reports)
	assert.Empty(t, back.Errors)
}

func TestLoadCollectionFile(t *testing.T) {
	orig := loadK8s(t)

	path := filepath.Join(t.TempDir(), "report.json")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, WriteCollection(orig, f))
	require.NoError(t, f.Close())

	back, err := LoadCollectionFile(path)
	require.NoError(t, err)
	require.True(t, proto.Equal(orig, back))

	_, err = LoadCollectionFile(filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
}

// TestLoadCollectionRejectsReducedReports is what stops a viewer from
// rendering a json-v2 file as "15 assets, no findings".
func TestLoadCollectionRejectsReducedReports(t *testing.T) {
	t.Run("json-v2 file", func(t *testing.T) {
		_, err := LoadCollectionFile(filepath.Join("testdata", "cnspec_report_pass.json"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reduced report")
	})

	t.Run("json-v1 shape", func(t *testing.T) {
		// json-v1 puts per-asset results under a top-level "data" key, which
		// ReportCollection does not have
		_, err := LoadCollection(strings.NewReader(`{"assets":{},"data":{}}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reduced report")
	})

	t.Run("garbage", func(t *testing.T) {
		_, err := LoadCollection(strings.NewReader("not json"))
		require.Error(t, err)
	})
}

// TestReporterJSONFullFormat exercises the wiring: the alias, the WriteReport
// switch, and PrintVulns.
func TestReporterJSONFullFormat(t *testing.T) {
	format, ok := Formats["json-full"]
	require.True(t, ok, "json-full alias is not registered")
	require.Equal(t, FormatJSONFull, format)
	assert.Contains(t, AllFormats(), "json-full")

	conf, err := ParseConfig("json-full")
	require.NoError(t, err)
	require.Equal(t, FormatJSONFull, conf.format)

	orig := loadUbuntu(t)

	var buf bytes.Buffer
	r := NewReporter(conf, false).WithOutput(&buf)
	require.NoError(t, r.WriteReport(context.Background(), orig))

	back, err := LoadCollection(&buf)
	require.NoError(t, err)
	require.True(t, proto.Equal(orig, back))

	// vuln reports are not report collections; the format must say so rather
	// than emit something a loader would choke on
	vulnErr := NewReporter(conf, false).WithOutput(&bytes.Buffer{}).PrintVulns(nil, "target")
	require.Error(t, vulnErr)
	assert.Contains(t, vulnErr.Error(), "json-full")
}

// TestJSONFullToFileTarget verifies --output-target routing works for the new
// format, since localFileHandler reuses WriteReport with a file writer.
func TestJSONFullToFileTarget(t *testing.T) {
	orig := loadK8s(t)

	conf, err := ParseConfig("json-full")
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "out.json")
	handler := &localFileHandler{file: path, conf: conf}
	require.NoError(t, handler.WriteReport(context.Background(), orig))

	back, err := LoadCollectionFile(path)
	require.NoError(t, err)
	require.True(t, proto.Equal(orig, back))
	assert.Len(t, back.Errors, 15)
}
