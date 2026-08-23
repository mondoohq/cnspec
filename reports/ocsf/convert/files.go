// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Writing the events out as one file per event class.

package convert

import (
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
)

// Encoding is how the per-class files of ConvertToDir are written.
type Encoding byte

const (
	// EncodingJSON writes newline-delimited JSON, one .jsonl per class.
	EncodingJSON Encoding = iota + 1
	// EncodingParquet writes Apache Parquet, one .parquet per class. Parquet has
	// no single-file form here: it is binary and its schema is per-class, so a
	// directory is the only shape it comes in.
	EncodingParquet
)

func (e Encoding) parquet() bool { return e == EncodingParquet }

func (e Encoding) ext() string {
	if e.parquet() {
		return ".parquet"
	}
	return ".jsonl"
}

// allExtensions is every extension ConvertToDir can produce, which is what a
// stale sweep has to look for whichever encoding this run used.
var allExtensions = []string{EncodingJSON.ext(), EncodingParquet.ext()}

// ConvertToDir writes a scan into dir as one file per OCSF event class and
// returns the paths written.
//
// A security data lake wants one event class per object: Amazon Security Lake
// requires it of custom sources, and it is what lets a Glue crawler derive a
// single schema per table.
//
// The events are streamed rather than collected: the converter produces one
// asset at a time and the writers encode and drop each set, so a fleet scan does
// not have to fit in memory twice.
func ConvertToDir(r *policy.ReportCollection, dir string, opts Options, enc Encoding) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errors.Wrapf(err, "failed to create the output directory %q", dir)
	}

	files := &classFiles{dir: dir, enc: enc}
	if err := Stream(r, files, opts); err != nil {
		return nil, err
	}
	if len(files.written) == 0 {
		log.Warn().Str("dir", dir).Msg("the scan produced no OCSF events, no files written")
	}
	return files.written, files.removeStale()
}

// classFiles routes each event class to its own file, creating one the first
// time that class produces an event. Creating them up front would leave empty
// files for the classes a scan happens not to produce.
type classFiles struct {
	dir     string
	enc     Encoding
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

	path := filepath.Join(c.dir, class+c.enc.ext())
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	var writer ocsf.Writer
	if c.enc.parquet() {
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
		for _, ext := range allExtensions {
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
	for _, class := range slices.Sorted(maps.Keys(c.writers)) {
		errs = multierror.Append(errs, c.writers[class].Close())
		errs = multierror.Append(errs, c.files[class].Close())
	}
	return errs.ErrorOrNil()
}
