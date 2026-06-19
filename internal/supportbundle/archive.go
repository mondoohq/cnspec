// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package supportbundle

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
)

// archiveDir writes a gzip-compressed tarball of srcDir to destPath. Entries
// are stored relative to srcDir's parent, so the archive unpacks into a single
// top-level directory whose name matches the bundle directory (e.g.
// cnspec-support-bundle-<ts>/manifest.json).
//
// Only regular files and directories are archived; symlinks and other special
// files are skipped (the bundle never writes them, and we don't want to follow
// links out of the tree).
func archiveDir(srcDir, destPath string) (retErr error) {
	out, err := os.Create(destPath)
	if err != nil {
		return errors.Wrap(err, "failed to create archive file")
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && retErr == nil {
			retErr = errors.Wrap(cerr, "failed to close archive file")
		}
	}()

	gz := gzip.NewWriter(out)
	defer func() {
		if cerr := gz.Close(); cerr != nil && retErr == nil {
			retErr = errors.Wrap(cerr, "failed to close gzip writer")
		}
	}()

	tw := tar.NewWriter(gz)
	defer func() {
		if cerr := tw.Close(); cerr != nil && retErr == nil {
			retErr = errors.Wrap(cerr, "failed to close tar writer")
		}
	}()

	// base is the bundle's parent dir; relative names therefore include the
	// bundle directory name as their first path component.
	base := filepath.Dir(srcDir)

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip anything that isn't a regular file or directory.
		if !info.Mode().IsRegular() && !info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(base, path)
		if err != nil {
			return errors.Wrapf(err, "failed to compute archive path for %s", path)
		}
		name := filepath.ToSlash(rel)

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return errors.Wrapf(err, "failed to build tar header for %s", path)
		}
		hdr.Name = name
		if info.IsDir() {
			hdr.Name += "/"
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return errors.Wrapf(err, "failed to write tar header for %s", name)
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return errors.Wrapf(err, "failed to open %s", path)
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return errors.Wrapf(err, "failed to copy %s into archive", name)
		}
		return nil
	})
}
