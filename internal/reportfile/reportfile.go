// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package reportfile opens the files a report format writes to disk.
//
// Every format writes the same kind of content -- cloud account ids and ARNs, the
// MQL source of every check, rendered assessments carrying the observed values,
// and the raw output of data queries -- into a path the user chose, from a scan
// that is routinely running as root. That makes two properties non-negotiable and
// identical across formats, which is why they live here rather than in each one:
//
//   - the file is created 0600, so it is not readable by every local account;
//   - the open does not follow a symlink, so a link pre-placed at an output path
//     cannot redirect the write onto a file outside the output directory and have
//     it truncated and overwritten with the findings.
//
// Report filenames are predictable by design (OCSF uses eight fixed class names,
// OHDF derives the name from the asset), so pre-placing that link needs no race
// and no guesswork -- only write access to the output directory.
package reportfile

import "os"

// Create opens path for writing, truncating an existing regular file, and fails
// rather than following a symlink. Use it for every file a report format writes.
func Create(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|oNoFollow, 0o600)
}
