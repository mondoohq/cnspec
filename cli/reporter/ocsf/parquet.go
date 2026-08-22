// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"io"

	"github.com/cockroachdb/errors"
	"github.com/parquet-go/parquet-go"
)

const (
	// maxDataPageSize caps an uncompressed data page at 1 MB, which is what
	// Amazon Security Lake asks custom sources for.
	maxDataPageSize = 1 << 20

	// maxRowsPerRowGroup keeps a row group well below Security Lake's 256 MB
	// (compressed) ceiling. An OCSF finding row is a few KB at most, so 50k rows
	// is far under the limit even before compression, and it keeps the writer's
	// memory bounded on very large scans.
	maxRowsPerRowGroup = 50_000
)

// WriteParquetClass writes the events of a single class as an Apache Parquet
// file. One class per file is a hard requirement for Security Lake custom
// sources, and it is what lets a reader derive a single schema for the file.
//
// The file is zstd-compressed (Security Lake's preferred codec) and its rows are
// written in the order they are held, so call Sort first to get them ordered by
// time.
func (e *Events) WriteParquetClass(class string, out io.Writer) error {
	switch class {
	case ClassComplianceFinding:
		return writeParquet(out, class, e.ComplianceFindings)
	case ClassVulnerabilityFinding:
		return writeParquet(out, class, e.VulnerabilityFindings)
	case ClassInventoryInfo:
		return writeParquet(out, class, e.InventoryInfos)
	default:
		return errors.Newf("unknown OCSF event class %q", class)
	}
}

func writeParquet[T any](out io.Writer, class string, rows []T) error {
	w := parquet.NewGenericWriter[T](out,
		parquet.Compression(&parquet.Zstd),
		parquet.PageBufferSize(maxDataPageSize),
		parquet.MaxRowsPerRowGroup(maxRowsPerRowGroup),
		parquet.KeyValueMetadata("ocsf.class_name", class),
	)

	if _, err := w.Write(rows); err != nil {
		return errors.Wrapf(err, "failed to write OCSF %s rows", class)
	}
	// Close writes the row group and the file footer; a file that is not closed
	// is not a parquet file.
	return errors.Wrapf(w.Close(), "failed to finalize the OCSF %s parquet file", class)
}
