// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/cli/reporter/hdf"
	"go.mondoo.com/cnspec/policy"
)

type localFileHandler struct {
	file string
	conf *PrintConfig
}

// we reuse the already implemented Reporter's WriteReport method by simply pointing the writer
// towards a file instead of stdout
func (h *localFileHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	trimmedFile := strings.TrimPrefix(h.file, "file://")
	f, err := os.Create(trimmedFile)
	if err != nil {
		return err
	}
	defer f.Close() //nolint: errcheck
	reporter := NewReporter(h.conf, false)
	reporter.out = f
	err = reporter.WriteReport(ctx, report)
	if err != nil {
		return err
	}
	log.Info().Str("file", trimmedFile).Msg("wrote report to file")
	return nil
}

// hdfDirHandler writes one OHDF document per scanned asset into a directory. It is
// selected when --output-target names a directory and the format is hdf, because an
// OHDF document describes a single target: consumers resolve a document down to one
// root profile, so several assets in one file would lose all but the first.
type hdfDirHandler struct {
	dir string
}

func (h *hdfDirHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	files, err := hdf.ConvertToDir(report, h.dir)
	if err != nil {
		return err
	}
	log.Info().Str("dir", h.dir).Int("files", len(files)).Msg("wrote OHDF reports to directory")
	return nil
}
