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
	// 0700, not 0755: these files carry cloud account ids, ARNs, the MQL source of
	// every check and its rendered assessment with the observed values in it, and
	// with Options.IncludeData the raw result of every data query. A scan is
	// routinely run as root, and a world-readable directory under it hands all of
	// that to every local account.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, errors.Wrapf(err, "failed to create the output directory %q", dir)
	}

	files := &classFiles{dir: dir, enc: enc}
	if err := Stream(r, files, opts); err != nil {
		return nil, err
	}

	// A run that wrote nothing does not sweep. removeStale deletes every name
	// cnspec knows, so on an empty conversion it would delete the whole of the
	// previous run's output and return nil -- a scan that discovered no assets
	// would erase a good report and exit 0, leaving "no findings" and "all clean"
	// indistinguishable. The sweep exists to stop two runs' files being counted
	// together; with no files of its own this run cannot cause that, so there is
	// nothing for it to fix and everything for it to destroy. Skipping is chosen
	// over erroring because an empty scan is not itself a failure -- the warning
	// is what says the directory still holds an older run.
	if len(files.written) == 0 {
		log.Warn().Str("dir", dir).
			Msg("the scan produced no OCSF events; no files written and the directory left as it was")
		return files.written, nil
	}
	files.removeStale()
	return files.written, nil
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
	// Not os.Create: the eight class filenames are fixed and public, so anyone who
	// can write the output directory can pre-place a symlink at one of them and
	// have a scan -- frequently running as root -- truncate and overwrite its
	// target with the findings. O_NOFOLLOW makes the open fail on a symlink
	// instead of following it (see nofollow_unix.go). The mode is 0600 for the
	// same reason the directory is 0700: the file carries account ids, MQL
	// source, observed values and, under IncludeData, the raw output of every
	// data query.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|oNoFollow, 0o600)
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

// removeStale deletes the per-class files this run did not write. It reports
// nothing: see the warning below for why a failure here is not the scan's.
//
// A run truncates only the files it opens, and writerFor opens one only for a
// class that produced an event. A directory reused across runs therefore
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
func (c *classFiles) removeStale() {
	written := make(map[string]bool, len(c.written))
	for _, path := range c.written {
		written[path] = true
	}

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
				// A warning, not an error. Every file of this run is already
				// written and closed by the time the sweep runs, so failing here
				// would turn a complete scan into a non-zero exit over a leftover
				// the process does not own -- a root-written file from an earlier
				// run is the ordinary way to get one. The warning names the file
				// so the double-counting risk it leaves behind is visible.
				log.Warn().Err(err).Str("file", path).
					Msg("could not remove a stale OCSF file from an earlier run; a consumer reading this directory may count its events too")
			}
		}
	}
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
