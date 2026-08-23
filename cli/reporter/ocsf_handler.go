// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
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
//
// Events are streamed rather than collected: the converter produces one asset at
// a time and the writers encode and drop each set, so a fleet scan does not have
// to fit in memory twice.
type ocsfFileHandler struct {
	target string
	conf   *PrintConfig
}

func (h *ocsfFileHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	isParquet := h.conf.format == FormatOcsfParquet
	target := strings.TrimPrefix(h.target, "file://")

	if !isParquet && !isDirTarget(target) {
		return h.writeSingleFile(target, report)
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		return errors.Wrapf(err, "failed to create the output directory %q", target)
	}

	files := &classFiles{dir: target, parquet: isParquet}
	if err := StreamOCSF(report, h.conf, files); err != nil {
		return err
	}
	if len(files.written) == 0 {
		log.Warn().Str("dir", target).Msg("the scan produced no OCSF events, no files written")
	}
	return files.removeStale()
}

func (h *ocsfFileHandler) writeSingleFile(target string, report *policy.ReportCollection) error {
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close() //nolint: errcheck

	if err := StreamOCSF(report, h.conf, ocsf.NewJSONWriter(f)); err != nil {
		return err
	}
	log.Info().Str("file", target).Msg("wrote OCSF report to file")
	return nil
}

// classFiles routes each event class to its own file, creating one the first
// time that class produces an event. Creating them up front would leave empty
// files for the classes a scan happens not to produce.
type classFiles struct {
	dir     string
	parquet bool
	files   map[string]*os.File
	writers map[string]ocsf.Writer
	written []string
}

func (c *classFiles) Write(events *ocsf.Events) error {
	for _, class := range events.Classes() {
		writer, err := c.writerFor(class)
		if err != nil {
			return err
		}
		if err := writer.Write(events); err != nil {
			return err
		}
	}
	return nil
}

func (c *classFiles) writerFor(class string) (ocsf.Writer, error) {
	if writer, ok := c.writers[class]; ok {
		return writer, nil
	}

	ext := ".jsonl"
	if c.parquet {
		ext = ".parquet"
	}
	path := filepath.Join(c.dir, class+ext)

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	var writer ocsf.Writer
	if c.parquet {
		writer, err = ocsf.NewParquetClassWriter(class, f)
		if err != nil {
			_ = f.Close()
			return nil, err
		}
	} else {
		writer = ocsf.NewJSONClassWriter(class, f)
	}

	if c.files == nil {
		c.files = map[string]*os.File{}
		c.writers = map[string]ocsf.Writer{}
	}
	c.files[class] = f
	c.writers[class] = writer
	c.written = append(c.written, path)

	log.Info().Str("file", path).Str("class", class).Msg("writing OCSF events to file")
	return writer, nil
}

// removeStale deletes the per-class files this run did not write.
//
// os.Create truncates only the files a run opens, and writerFor opens one only
// for a class that produced an event. A directory reused across runs therefore
// keeps whatever the previous run left: switching -o ocsf-json,ocsf-findings from
// detection to compliance is the ordinary way to get there, and it leaves the old
// detection_finding.jsonl sitting next to the new compliance_finding.jsonl. A
// crawler pointed at the directory reads both and counts every check twice --
// exactly what reporting each check in one class exists to prevent. Switching
// between the JSON and Parquet flavors has the same effect, so both extensions
// are swept.
//
// Only the names cnspec itself writes are considered, so anything else in the
// directory is left alone.
func (c *classFiles) removeStale() error {
	written := make(map[string]bool, len(c.written))
	for _, path := range c.written {
		written[path] = true
	}

	var errs *multierror.Error
	for _, class := range ocsf.AllClasses() {
		for _, ext := range []string{".jsonl", ".parquet"} {
			path := filepath.Join(c.dir, class+ext)
			if written[path] {
				continue
			}
			switch err := os.Remove(path); {
			case err == nil:
				log.Info().Str("file", path).Msg("removed a stale OCSF file from an earlier run")
			case !os.IsNotExist(err):
				errs = multierror.Append(errs, err)
			}
		}
	}
	return errs.ErrorOrNil()
}

// Close finalizes every file it opened. Every one is closed even if an earlier
// one fails, so a single bad file does not leave the rest without their footers.
func (c *classFiles) Close() error {
	var errs *multierror.Error
	for _, class := range sortedKeys(c.writers) {
		errs = multierror.Append(errs, c.writers[class].Close())
		errs = multierror.Append(errs, c.files[class].Close())
	}
	return errs.ErrorOrNil()
}
