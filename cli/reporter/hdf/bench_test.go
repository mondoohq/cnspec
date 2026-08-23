// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package hdf

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/rs/zerolog"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/utils/iox"
)

// benchCollection grows the recorded ubuntu scan to the requested number of assets
// by cloning its report onto extra MRNs, so the per-asset paths have a fleet to
// work on without a 2.4 MB fixture per asset.
func benchCollection(b testing.TB, assets int) *policy.ReportCollection {
	b.Helper()
	raw, err := os.ReadFile("../testdata/report-ubuntu.json")
	if err != nil {
		b.Fatal(err)
	}
	var r policy.ReportCollection
	if err := json.Unmarshal(raw, &r); err != nil {
		b.Fatal(err)
	}
	var srcMrn string
	for k := range r.Assets {
		srcMrn = k
	}
	src, srcRep, srcRes := r.Assets[srcMrn], r.Reports[srcMrn], r.ResolvedPolicies[srcMrn]
	for i := 0; i < assets-1; i++ {
		m := fmt.Sprintf("%s-%d", srcMrn, i)
		r.Assets[m] = &inventory.Asset{Name: fmt.Sprintf("host-%d", i), Platform: src.Platform}
		r.Reports[m] = srcRep
		r.ResolvedPolicies[m] = srcRes
	}
	return &r
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func BenchmarkConvertToHDF(b *testing.B) {
	// The multi-asset path logs a warning per call; it would dominate the output.
	restore := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	b.Cleanup(func() { zerolog.SetGlobalLevel(restore) })

	for _, n := range []int{1, 200} {
		r := benchCollection(b, n)
		b.Run(fmt.Sprintf("assets=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			w := iox.IOWriter{Writer: discard{}}
			for i := 0; i < b.N; i++ {
				if err := Convert(r, &w); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkConvertToHDFDir(b *testing.B) {
	r := benchCollection(b, 200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dir := b.TempDir()
		if _, err := ConvertToDir(r, dir); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHDFDocuments(b *testing.B) {
	r := benchCollection(b, 200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := hdfDocuments(r); err != nil {
			b.Fatal(err)
		}
	}
}
