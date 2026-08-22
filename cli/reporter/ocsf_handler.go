// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/cli/reporter/ocsf"
	"go.mondoo.com/cnspec/policy"
)

// ocsfFileHandler writes OCSF events to the local filesystem.
//
// A security data lake wants one event class per object: Amazon Security Lake
// requires it of custom sources, and it is what lets a Glue crawler derive a
// single schema per table. So whenever the target is a directory, each class
// goes into its own file. A plain file target keeps all classes in one
// newline-delimited JSON stream, which is what a SIEM ingesting a single file
// expects. Parquet has no such single-file mode: it is binary and per-class, so
// it always needs a directory.
type ocsfFileHandler struct {
	target string
	conf   *PrintConfig
}

func (h *ocsfFileHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	events, err := ConvertToOCSF(report, h.conf)
	if err != nil {
		return err
	}

	isParquet := h.conf.format == FormatOcsfParquet
	target := strings.TrimPrefix(h.target, "file://")

	if !isParquet && !isDirTarget(target) {
		return h.writeSingleFile(target, events)
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		return errors.Wrapf(err, "failed to create the output directory %q", target)
	}

	classes := events.Classes()
	if len(classes) == 0 {
		log.Warn().Str("dir", target).Msg("the scan produced no OCSF events, no files written")
		return nil
	}

	for _, class := range classes {
		if err := h.writeClass(target, class, events, isParquet); err != nil {
			return err
		}
	}
	return nil
}

func (h *ocsfFileHandler) writeSingleFile(target string, events *ocsf.Events) error {
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close() //nolint: errcheck

	if err := events.WriteJSON(f); err != nil {
		return err
	}
	log.Info().Str("file", target).Int("events", events.Len()).Msg("wrote OCSF report to file")
	return nil
}

func (h *ocsfFileHandler) writeClass(dir, class string, events *ocsf.Events, isParquet bool) error {
	ext := ".jsonl"
	if isParquet {
		ext = ".parquet"
	}
	path := filepath.Join(dir, class+ext)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint: errcheck

	if isParquet {
		err = events.WriteParquetClass(class, f)
	} else {
		err = events.WriteJSONClass(class, f)
	}
	if err != nil {
		return err
	}

	log.Info().Str("file", path).Str("class", class).Msg("wrote OCSF events to file")
	return nil
}

// isDirTarget reports whether the target names a directory: one that already
// exists, or a path written as a directory (a trailing separator, or no file
// extension).
func isDirTarget(target string) bool {
	if target == "" {
		return false
	}
	if strings.HasSuffix(target, string(os.PathSeparator)) || strings.HasSuffix(target, "/") {
		return true
	}
	if info, err := os.Stat(target); err == nil {
		return info.IsDir()
	}
	return filepath.Ext(target) == ""
}
