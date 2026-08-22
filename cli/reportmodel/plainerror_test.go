// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportmodel

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"go.mondoo.com/cnspec/policy"
)

// gRPC wraps a provider's own sentence in a transport envelope. The code is a
// detail of how the message travelled; what a reader can act on is the part
// after desc=. Every scan error in the fixtures carries the wrapper, so this is
// the normal case rather than an edge one.
func TestScanErrorsLoseTheirRPCEnvelope(t *testing.T) {
	raw, err := os.ReadFile("../reporter/testdata/report-k8s.json")
	if err != nil {
		t.Fatal(err)
	}
	var collection policy.ReportCollection
	if err := json.Unmarshal(raw, &collection); err != nil {
		t.Fatal(err)
	}

	wrapped := 0
	for _, msg := range collection.Errors {
		if strings.HasPrefix(msg, "rpc error:") {
			wrapped++
		}
	}
	if wrapped == 0 {
		t.Fatal("fixture no longer carries wrapped errors; this test proves nothing")
	}
	t.Logf("%d of %d fixture errors arrive wrapped", wrapped, len(collection.Errors))

	report := New(&collection)
	for _, asset := range report.Assets {
		if strings.HasPrefix(asset.ScanError, "rpc error:") {
			t.Errorf("%s: still wrapped: %q", asset.Name, asset.ScanError)
		}
		if asset.ScanError == "" {
			t.Errorf("%s: the reason was lost entirely", asset.Name)
		}
	}
}

func TestPlainErrorKeepsWhatItCannotImprove(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"rpc error: code = InvalidArgument desc = asset doesn't support any policies", "asset doesn't support any policies"},
		{"connection refused", "connection refused"},
		{"", ""},
		{"rpc error: malformed", "rpc error: malformed"},
	} {
		if got := plainError(tc.in); got != tc.want {
			t.Errorf("plainError(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
