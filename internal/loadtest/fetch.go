// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	cockroach "github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
)

const gcsScheme = "gs://"

// MaterializeSeeds resolves --input into a local directory of .db files. For
// local paths it's a no-op pass-through; for gs:// URIs it lists the bucket
// prefix, downloads every .db object to a fresh temp dir, and returns that
// dir along with a cleanup func the caller must defer.
//
// Authentication for GCS uses Application Default Credentials. Operators
// configure auth via `gcloud auth application-default login`,
// $GOOGLE_APPLICATION_CREDENTIALS, or workload identity.
func MaterializeSeeds(ctx context.Context, input string) (string, func(), error) {
	if !strings.HasPrefix(input, gcsScheme) {
		// Local path. Trust the caller; LoadTemplates will surface a real
		// error if it doesn't exist.
		return input, func() {}, nil
	}

	bucket, prefix, err := parseGCSURI(input)
	if err != nil {
		return "", func() {}, err
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", func() {}, cockroach.Wrap(err,
			"connect to GCS (try `gcloud auth application-default login` or set GOOGLE_APPLICATION_CREDENTIALS)")
	}
	defer client.Close()

	tmpDir, err := os.MkdirTemp("", "cnspec-loadtest-seeds-*")
	if err != nil {
		return "", func() {}, cockroach.Wrap(err, "create seed staging dir")
	}
	cleanup := func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Warn().Err(err).Str("dir", tmpDir).Msg("failed to remove seed staging dir")
		}
	}

	count, err := downloadDBObjects(ctx, client.Bucket(bucket), prefix, tmpDir)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	if count == 0 {
		cleanup()
		return "", func() {}, fmt.Errorf("no .db files found under %s", input)
	}

	log.Info().Int("count", count).Str("uri", input).Str("dir", tmpDir).Msg("fetched seeds from GCS")
	return tmpDir, cleanup, nil
}

// parseGCSURI splits gs://bucket/prefix into bucket and prefix. The bucket
// is required; the prefix is optional and may end in / (treated identically
// to no slash, like gsutil).
func parseGCSURI(uri string) (bucket, prefix string, err error) {
	trimmed := strings.TrimPrefix(uri, gcsScheme)
	if trimmed == "" {
		return "", "", fmt.Errorf("invalid GCS URI %q: missing bucket", uri)
	}
	parts := strings.SplitN(trimmed, "/", 2)
	bucket = parts[0]
	if bucket == "" {
		return "", "", fmt.Errorf("invalid GCS URI %q: empty bucket", uri)
	}
	if len(parts) == 2 {
		prefix = parts[1]
	}
	return bucket, prefix, nil
}

// downloadDBObjects iterates objects under prefix and saves any matching
// *.db to tmpDir. Returns the number successfully downloaded. Failures on
// individual objects abort early — partial seed sets produce skewed load.
func downloadDBObjects(ctx context.Context, bkt *storage.BucketHandle, prefix, tmpDir string) (int, error) {
	it := bkt.Objects(ctx, &storage.Query{Prefix: prefix})
	count := 0
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return count, cockroach.Wrap(err, "list GCS objects")
		}
		if !strings.HasSuffix(attrs.Name, ".db") {
			continue
		}
		// Flatten any in-bucket directory structure into tmpDir; collisions
		// across nested folders are unlikely in practice (seed names tend
		// to be unique hashes) and we'd rather fail loudly than silently
		// drop overlaps.
		dest := filepath.Join(tmpDir, filepath.Base(attrs.Name))
		if err := downloadObject(ctx, bkt, attrs.Name, dest); err != nil {
			return count, cockroach.Wrapf(err, "download %s", attrs.Name)
		}
		count++
	}
	return count, nil
}

func downloadObject(ctx context.Context, bkt *storage.BucketHandle, name, dest string) error {
	rc, err := bkt.Object(name).NewReader(ctx)
	if err != nil {
		return err
	}
	defer rc.Close()

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return err
	}
	return nil
}
