// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

type fileBundleResolver struct{}

func defaultFileBundleResolver() *fileBundleResolver {
	return NewFileBundleResolver()
}

func (l *fileBundleResolver) Load(ctx context.Context, path string) (*Bundle, error) {
	return loadBundlesFromPaths(path)
}

func NewFileBundleResolver() *fileBundleResolver {
	return &fileBundleResolver{}
}

func (r *fileBundleResolver) IsApplicable(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// loadBundlesFromPaths loads a single policy bundle file or a bundle that
// was split into multiple files into a single PolicyBundle struct
func loadBundlesFromPaths(paths ...string) (*Bundle, error) {
	// load all the source files
	resolvedFilenames, err := WalkPolicyBundleFiles(paths...)
	if err != nil {
		log.Error().Err(err).Msg("could not resolve bundle files")
		return nil, err
	}

	// aggregate all files into a single policy bundle
	aggregatedBundle, err := aggregateFilesToBundle(resolvedFilenames)
	if err != nil {
		log.Debug().Err(err).Msg("could not merge bundle files")
		return nil, err
	}

	return aggregatedBundle, nil
}

// WalkPolicyBundleFiles iterates over all provided filenames and
// checks if the name is a file or a directory. If the filename
// is a directory, it walks the directory recursively
func WalkPolicyBundleFiles(filenames ...string) ([]string, error) {
	// resolve file names
	resolvedFilenames := []string{}
	for i := range filenames {
		filename := filenames[i]
		fi, err := os.Stat(filename)
		if err != nil {
			return nil, errors.Wrap(err, "could not load policy bundle file: "+filename)
		}

		if fi.IsDir() {
			err := filepath.WalkDir(filename, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				// we ignore directories because WalkDir already walks them
				if d.IsDir() {
					return nil
				}

				// only consider .yaml|.yml files
				if strings.HasSuffix(d.Name(), ".mql.yaml") || strings.HasSuffix(d.Name(), ".mql.yml") {
					resolvedFilenames = append(resolvedFilenames, path)
				}

				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			resolvedFilenames = append(resolvedFilenames, filename)
		}
	}

	return resolvedFilenames, nil
}

// aggregateFilesToBundle iterates over all provided files and loads their content.
// It assumes that all provided files are checked upfront and are not a directory
func aggregateFilesToBundle(paths []string) (*Bundle, error) {
	// iterate over all files, load them and merge them
	mergedBundle := &Bundle{}

	for i := range paths {
		path := paths[i]
		log.Debug().Str("path", path).Msg("local>loading policy bundle file")
		bundle, err := bundleFromSingleFile(path)
		if err != nil {
			return nil, errors.Wrap(err, "could not load file: "+path)
		}

		mergedBundle = Merge(mergedBundle, bundle)
	}

	return mergedBundle, nil
}

// bundleFromSingleFile loads a policy bundle from a single file
func bundleFromSingleFile(path string) (*Bundle, error) {
	bundleData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return BundleFromYAML(bundleData)
}
