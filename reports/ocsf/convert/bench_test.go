// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"bytes"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// largeReportCollection builds a scan of assets x checks, sized like a fleet
// scan rather than the small fixtures the correctness tests use.
func largeReportCollection(assets, checks int) *policy.ReportCollection {
	res := &policy.ReportCollection{
		Assets:           map[string]*inventory.Asset{},
		Reports:          map[string]*policy.Report{},
		ResolvedPolicies: map[string]*policy.ResolvedPolicy{},
		Bundle:           &policy.Bundle{},
	}

	reporting := map[string]*policy.StringArray{}
	for c := range checks {
		codeID := "code-" + strconv.Itoa(c)
		reporting[codeID] = nil
		res.Bundle.Queries = append(res.Bundle.Queries, &policy.Mquery{
			Mrn:    "//policy.api.mondoo.app/queries/check-" + strconv.Itoa(c),
			CodeId: codeID,
			Title:  "Ensure the thing number " + strconv.Itoa(c) + " is configured correctly",
			Mql:    "sshd.config.params['PermitRootLogin'] == \"no\" && file('/etc/thing').permissions.mode == 0644",
			Docs: &policy.MqueryDocs{
				Desc:        "A description of what this check verifies and why it matters, of a length typical for the policies cnspec ships.",
				Remediation: &policy.Remediation{Items: []*policy.TypedDoc{{Id: "default", Desc: "Set the thing to the expected value."}}},
			},
			Tags: map[string]string{
				"compliance/cis-benchmark":  "1." + strconv.Itoa(c),
				"compliance/iso-27001-2022": "a-8-" + strconv.Itoa(c),
				"compliance/nist-sp-800-53": "ac-" + strconv.Itoa(c),
			},
		})
	}

	for a := range assets {
		mrn := "//assets.api.mondoo.app/spaces/bench/assets/" + strconv.Itoa(a)
		res.Assets[mrn] = &inventory.Asset{
			Name:        "host-" + strconv.Itoa(a),
			Mrn:         mrn,
			PlatformIds: []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-" + strconv.Itoa(a)},
			Platform: &inventory.Platform{
				Name: "ubuntu", Version: "22.04", Arch: "amd64", Kind: "virtualmachine",
				Family: []string{"debian", "linux", "unix", "os"}, Runtime: "aws-ec2-instance",
			},
			Labels: map[string]string{"env": "prod", "team": "platform"},
		}

		scores := make(map[string]*policy.Score, checks)
		for c := range checks {
			value := uint32(100)
			if c%3 == 0 {
				value = 0
			}
			scores["code-"+strconv.Itoa(c)] = &policy.Score{Type: policy.ScoreType_Result, Value: value}
		}
		res.Reports[mrn] = &policy.Report{ScoringMrn: mrn, EntityMrn: mrn, Scores: scores}
		res.ResolvedPolicies[mrn] = &policy.ResolvedPolicy{
			CollectorJob: &policy.CollectorJob{ReportingQueries: reporting},
		}
	}
	return res
}

func BenchmarkOcsfConvert(b *testing.B) {
	report := largeReportCollection(200, 50)
	conf := Options{Version: ocsf.DefaultVersion, Findings: ocsf.FindingsCompliance}

	b.ReportAllocs()
	for b.Loop() {
		events, err := convertAt(report, conf, fixedScanTime)
		if err != nil {
			b.Fatal(err)
		}
		if events.Len() != 200*50+200 {
			b.Fatalf("unexpected event count %d", events.Len())
		}
	}
}

func BenchmarkOcsfWriteJSON(b *testing.B) {
	report := largeReportCollection(200, 50)
	events, err := convertAt(report, Options{Version: ocsf.DefaultVersion, Findings: ocsf.FindingsCompliance}, fixedScanTime)
	require.NoError(b, err)

	b.ReportAllocs()
	for b.Loop() {
		if err := events.WriteJSON(io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOcsfWriteParquet(b *testing.B) {
	report := largeReportCollection(200, 50)
	events, err := convertAt(report, Options{Version: ocsf.DefaultVersion, Findings: ocsf.FindingsCompliance}, fixedScanTime)
	require.NoError(b, err)

	b.ReportAllocs()
	for b.Loop() {
		buf := bytes.Buffer{}
		if err := events.WriteParquetClass(ocsf.ClassComplianceFinding, &buf); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOcsfConvertBareChecks is the same shape with no docs, remediation or
// compliance tags, which isolates how much of the conversion is spent deriving
// per-check documentation once per asset.
func BenchmarkOcsfConvertBareChecks(b *testing.B) {
	report := largeReportCollection(200, 50)
	for _, query := range report.Bundle.Queries {
		query.Docs = nil
		query.Tags = nil
	}
	conf := Options{Version: ocsf.DefaultVersion, Findings: ocsf.FindingsCompliance}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := convertAt(report, conf, fixedScanTime); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOcsfStreamJSON is the streamed path: convert and write asset by
// asset, holding one asset's events at a time rather than the whole scan.
func BenchmarkOcsfStreamJSON(b *testing.B) {
	report := largeReportCollection(200, 50)
	conf := Options{}

	b.ReportAllocs()
	for b.Loop() {
		if err := Stream(report, ocsf.NewJSONWriter(io.Discard), conf); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOcsfStreamParquet is the streamed Parquet path.
//
// It exists because only the JSON path was benchmarked, and the streaming claim
// in ADR-0005 was then read as covering both. It does not: the Parquet writer
// buffers a row group and a page per column, so its footprint is bounded by those
// buffers rather than by one asset's events, and it measures two orders of
// magnitude above the JSON path on the same scan. Benchmarking both is what keeps
// that visible.
func BenchmarkOcsfStreamParquet(b *testing.B) {
	report := largeReportCollection(200, 50)
	conf := Options{}

	b.ReportAllocs()
	for b.Loop() {
		w, err := ocsf.NewParquetClassWriter(ocsf.ClassComplianceFinding, io.Discard)
		if err != nil {
			b.Fatal(err)
		}
		if err := Stream(report, w, conf); err != nil {
			b.Fatal(err)
		}
	}
}
