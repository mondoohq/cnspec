// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"io"

	"github.com/parquet-go/parquet-go"
)

const (
	// maxDataPageSize caps an uncompressed data page at 1 MB, which is what
	// Amazon Security Lake asks custom sources for.
	maxDataPageSize = 1 << 20

	// maxRowsPerRowGroup keeps a row group well below Security Lake's 256 MB
	// (compressed) ceiling. An OCSF finding row is a few KB at most, so 50k rows
	// is far under the limit even before compression.
	//
	// It bounds how many rows the writer buffers, but it is not what makes the
	// Parquet path's memory what it is, and lowering it is not the lever it looks
	// like: measured on a streamed scan, 5k rows per group moved peak heap from
	// 340 MB to 216 MB at 100k rows and made it worse at 10k. The floor is the
	// page buffers -- maxDataPageSize per column, and a finding class has many
	// columns -- which no row count reaches below. Unlike the JSON writer, which
	// holds one asset at a time, the Parquet path is bounded but not flat; see
	// ADR-0005.
	maxRowsPerRowGroup = 50_000
)

// newParquetWriter builds the Parquet writer for one event class: zstd, which is
// Security Lake's preferred codec, and page and row group sizes it accepts.
func newParquetWriter[T any](out io.Writer, class string) *parquet.GenericWriter[T] {
	return parquet.NewGenericWriter[T](out,
		parquet.Compression(&parquet.Zstd),
		parquet.PageBufferSize(maxDataPageSize),
		parquet.MaxRowsPerRowGroup(maxRowsPerRowGroup),
		parquet.KeyValueMetadata("ocsf.class_name", class),
	)
}

// WriteParquetClass writes the events of a single class as an Apache Parquet
// file in one shot. Streaming callers use NewParquetClassWriter instead.
func (e *Events) WriteParquetClass(class string, out io.Writer) error {
	w, err := NewParquetClassWriter(class, out)
	if err != nil {
		return err
	}
	if err := w.Write(e); err != nil {
		return err
	}
	return w.Close()
}
