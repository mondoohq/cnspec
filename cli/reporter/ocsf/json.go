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
	case ClassDetectionFinding:
		return encodeAll(enc, e.DetectionFindings)
	case ClassVulnerabilityFinding:
		return encodeAll(enc, e.VulnerabilityFindings)
	case ClassInventoryInfo:
		return encodeAll(enc, e.InventoryInfos)
	default:
		return errors.Newf("unknown OCSF event class %q", class)
	}
}

// EachJSON calls fn with the JSON encoding of every event, class by class in
// write order. A transport that wraps each event in an envelope of its own, such
// as Splunk HEC, iterates over this rather than over a byte stream.
func (e *Events) EachJSON(fn func(class string, event []byte) error) error {
	for _, class := range e.Classes() {
		var err error
		switch class {
		case ClassComplianceFinding:
			err = eachJSON(class, e.ComplianceFindings, fn)
		case ClassDetectionFinding:
			err = eachJSON(class, e.DetectionFindings, fn)
		case ClassVulnerabilityFinding:
			err = eachJSON(class, e.VulnerabilityFindings, fn)
		case ClassInventoryInfo:
			err = eachJSON(class, e.InventoryInfos, fn)
		default:
			err = errors.Newf("unknown OCSF event class %q", class)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func eachJSON[T any](class string, rows []T, fn func(string, []byte) error) error {
	for i := range rows {
		raw, err := json.Marshal(rows[i])
		if err != nil {
			return err
		}
		if err := fn(class, raw); err != nil {
			return err
		}
	}
	return nil
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
