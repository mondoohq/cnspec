// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reportfixture

import (
	_ "embed"
	"encoding/json"

	"go.mondoo.com/cnspec/policy"
)

// ubuntuScan is a recorded `cnspec scan` of one Ubuntu host: real check results,
// real vulnerability report, 2.4 MB of it. The hand-built fixtures above are what
// the outcome-by-outcome assertions run on; this is what proves a converter
// survives a scan it did not have written for it.
//
//go:embed testdata/report-ubuntu.json
var ubuntuScan []byte

// UbuntuScan decodes the recorded scan. It is embedded rather than read from a
// path because it is shared by the reporters in cli/reporter and in reports/...,
// and a relative path would have to be written differently in each of them - and
// re-written again the next time one of them moves.
func UbuntuScan() (*policy.ReportCollection, error) {
	res := &policy.ReportCollection{}
	if err := json.Unmarshal(ubuntuScan, res); err != nil {
		return nil, err
	}
	return res, nil
}

// UbuntuScanJSON is the raw bytes of the recorded scan, for the callers that
// decode it into something other than a ReportCollection.
func UbuntuScanJSON() []byte {
	return ubuntuScan
}
