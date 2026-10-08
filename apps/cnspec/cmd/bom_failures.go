// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"errors"
	"fmt"
	"sort"

	cr "go.mondoo.com/mql/cli/reporter"
)

// errBomGenerationFailed is returned when a bill of materials could not be
// generated for at least one asset.
var errBomGenerationFailed = errors.New("bill of materials generation failed")

// bomFailure describes an asset whose bill of materials could not be
// generated.
type bomFailure struct {
	Asset  string
	Reason string
}

// withoutFailedAssets splits the assets that could not be scanned (a
// connection that failed, a provider that could not start, ...) off a report,
// so no bill of materials is generated for them. Such an asset has no data,
// and its bill of materials would be a valid but empty document that is
// indistinguishable from an asset without any software. It returns the
// remaining report and one failure per removed asset, ordered by asset name.
// The input report is not modified.
func withoutFailedAssets(report *cr.Report) (*cr.Report, []bomFailure) {
	if report == nil || len(report.Errors) == 0 {
		return report, nil
	}

	res := &cr.Report{
		Assets: make(map[string]*cr.Asset, len(report.Assets)),
		Data:   make(map[string]*cr.DataValues, len(report.Data)),
		Errors: report.Errors,
	}
	for mrn, asset := range report.Assets {
		if _, failed := report.Errors[mrn]; !failed {
			res.Assets[mrn] = asset
		}
	}
	for mrn, data := range report.Data {
		if _, failed := report.Errors[mrn]; !failed {
			res.Data[mrn] = data
		}
	}

	failures := make([]bomFailure, 0, len(report.Errors))
	for mrn, scanErr := range report.Errors {
		name := report.Assets[mrn].GetName()
		if name == "" {
			name = mrn
		}
		if name == "" {
			name = "<unnamed asset>"
		}
		failures = append(failures, bomFailure{Asset: name, Reason: scanErr})
	}
	sort.Slice(failures, func(i, j int) bool {
		if failures[i].Asset != failures[j].Asset {
			return failures[i].Asset < failures[j].Asset
		}
		return failures[i].Reason < failures[j].Reason
	})
	return res, failures
}

// bomFailuresError turns the per-asset failures of a bill of materials run
// into one error, or nil when there are none. kind is "SBOM" or "AIBOM";
// generated is the number of bills of materials that were written.
//
// Like scan, any failed asset fails the command, so a partial result is never
// reported as a success. The bills of materials of the assets that did
// succeed are still written.
func bomFailuresError(kind string, failures []bomFailure, generated int) error {
	if len(failures) == 0 {
		return nil
	}

	msg := fmt.Sprintf("could not generate the %s for %d of %d asset(s)", kind, len(failures), len(failures)+generated)
	for _, f := range failures {
		msg += fmt.Sprintf("\n  - asset %q: %s", f.Asset, f.Reason)
	}
	return fmt.Errorf("%w: %s", errBomGenerationFailed, msg)
}
