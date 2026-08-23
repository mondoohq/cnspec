// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"io"

	"github.com/cockroachdb/errors"
)

// Writer consumes events as they are produced.
//
// A scan is converted asset by asset, so a writer receives many small sets
// rather than one large one. That is what keeps peak memory tied to the size of
// an asset rather than to the size of the whole report: each set can be encoded
// and dropped. Close finishes the encoding, which for Parquet means writing the
// footer without which the file is not a Parquet file.
type Writer interface {
	Write(*Events) error
	Close() error
}

// NewJSONWriter writes every class as newline-delimited JSON to one stream.
func NewJSONWriter(out io.Writer) Writer {
	return &jsonWriter{out: out}
}

// NewJSONClassWriter writes only the events of one class, for a destination that
// keeps a stream per class.
func NewJSONClassWriter(class string, out io.Writer) Writer {
	return &jsonWriter{out: out, class: class}
}

type jsonWriter struct {
	out   io.Writer
	class string
}

func (w *jsonWriter) Write(events *Events) error {
	if w.class == "" {
		return events.WriteJSON(w.out)
	}
	return events.WriteJSONClass(w.class, w.out)
}

func (w *jsonWriter) Close() error { return nil }

// NewParquetClassWriter writes the events of one class as a Parquet file. One
// class per file is a hard requirement for Security Lake custom sources, and it
// is what lets a reader derive a single schema for the file.
//
// Rows are written in the order they arrive, so the caller decides the ordering;
// Security Lake asks for events ordered by time.
func NewParquetClassWriter(class string, out io.Writer) (Writer, error) {
	switch class {
	case ClassComplianceFinding:
		return classWriter(class, out, func(e *Events) []ComplianceFinding { return e.ComplianceFindings }), nil
	case ClassDetectionFinding:
		return classWriter(class, out, func(e *Events) []DetectionFinding { return e.DetectionFindings }), nil
	case ClassVulnerabilityFinding:
		return classWriter(class, out, func(e *Events) []VulnerabilityFinding { return e.VulnerabilityFindings }), nil
	case ClassInventoryInfo:
		return classWriter(class, out, func(e *Events) []InventoryInfo { return e.InventoryInfos }), nil
	default:
		return nil, errors.Newf("unknown OCSF event class %q", class)
	}
}

// classWriter ties a generic Parquet writer to the field of Events it drains,
// which is what lets one Writer interface cover all four classes.
func classWriter[T any](class string, out io.Writer, rows func(*Events) []T) Writer {
	w := newParquetWriter[T](out, class)
	return &parquetWriter{
		class: class,
		write: func(events *Events) error {
			batch := rows(events)
			if len(batch) == 0 {
				return nil
			}
			_, err := w.Write(batch)
			return errors.Wrapf(err, "failed to write OCSF %s rows", class)
		},
		close: func() error {
			// Close writes the row group and the file footer; a file that is not
			// closed is not a parquet file.
			return errors.Wrapf(w.Close(), "failed to finalize the OCSF %s parquet file", class)
		},
	}
}

type parquetWriter struct {
	class string
	write func(*Events) error
	close func() error
}

func (w *parquetWriter) Write(events *Events) error { return w.write(events) }

func (w *parquetWriter) Close() error { return w.close() }
