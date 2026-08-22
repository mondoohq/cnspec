// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"encoding/json"
	"io"

	"github.com/cockroachdb/errors"
)

// WriteJSON writes every event as newline-delimited JSON, one event per line,
// classes in write order. ND-JSON is what OCSF consumers expect: every record
// stands on its own and names its class in `class_uid`.
func (e *Events) WriteJSON(out io.Writer) error {
	for _, class := range e.Classes() {
		if err := e.WriteJSONClass(class, out); err != nil {
			return err
		}
	}
	return nil
}

// WriteJSONClass writes the events of a single class as newline-delimited JSON.
func (e *Events) WriteJSONClass(class string, out io.Writer) error {
	enc := json.NewEncoder(out)
	switch class {
	case ClassComplianceFinding:
		return encodeAll(enc, e.ComplianceFindings)
	case ClassVulnerabilityFinding:
		return encodeAll(enc, e.VulnerabilityFindings)
	case ClassInventoryInfo:
		return encodeAll(enc, e.InventoryInfos)
	default:
		return errors.Newf("unknown OCSF event class %q", class)
	}
}

func encodeAll[T any](enc *json.Encoder, rows []T) error {
	for i := range rows {
		// Encode terminates every value with a newline, which is the record
		// separator we want.
		if err := enc.Encode(rows[i]); err != nil {
			return err
		}
	}
	return nil
}
