// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"
	"path/filepath"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// logFileKey is the mondoo.yml key, bound to `cnspec serve --log-file`, that
// sends the service's log output to a file instead of stderr. Under systemd,
// stderr goes to the journal, and from there rsyslog copies it into
// /var/log/messages or /var/log/syslog.
const logFileKey = "log_file"

// configureServeLogging selects the writer for `cnspec serve`. The log level
// has already been applied by initLogger, from DEBUG/TRACE, a logging
// configuration file, or log-level.
//
// When logFile is set, the process's stderr is redirected to it at the file
// descriptor level rather than by swapping the zerolog writer. Two outputs
// bypass the zerolog logger and would otherwise still reach the journal: the
// raw stderr of provider plugins, which the provider coordinator copies to
// logger.LogOutputWriter (a wrapper around os.Stderr), and Go runtime panics,
// which the runtime writes to file descriptor 2 directly.
//
// If the file cannot be opened, the service logs the error and keeps logging
// to stderr. A service that scans and logs to the journal is preferable to one
// that refuses to start over its log destination.
func configureServeLogging(logFile string, hasLoggingConfig bool) {
	var redirectErr error
	if logFile != "" {
		redirectErr = redirectFD(logFile, int(os.Stderr.Fd()))
	}
	toFile := logFile != "" && redirectErr == nil

	// A logging configuration file has already installed its writer in
	// initLogger. Its "cli" writer writes to os.Stderr, so it follows the
	// redirect above.
	if !hasLoggingConfig {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
			// ANSI color codes are noise in a file a SIEM or grep reads.
			NoColor: toFile,
		})
	}

	switch {
	case logFile == "":
	case redirectErr != nil:
		log.Error().Err(redirectErr).Str("path", logFile).Msg("could not write service logs to file, logging to stderr")
	default:
		log.Info().Str("path", logFile).Msg("writing service logs to file")
	}
}

// openLogFile opens path for appending, creating it and its parent directory
// if needed. Service logs can name assets and, at debug or trace level,
// include request details, so neither is readable by other users.
//
// O_APPEND makes every write land at the current end of the file, so an
// external rotator can truncate it in place (logrotate's copytruncate) without
// leaving the next write at a stale offset.
func openLogFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, errors.Wrap(err, "could not create log directory")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o640)
	if err != nil {
		return nil, errors.Wrap(err, "could not open log file")
	}
	return f, nil
}
