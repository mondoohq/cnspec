// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scandb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// Template is the in-memory representation of one scan database, read once at
// startup and reused as the source-of-truth for synthesizing assets and scans.
type Template struct {
	Path          string
	Asset         *inventory.Asset
	Scores        []*policy.Score
	Data          map[string]*llx.Result
	Risks         []*policy.ScoredRiskFactor
	FilterCodeIDs []string
}

// LoadTemplates walks dir non-recursively, opens every *.db file as a scan
// database, and returns its full contents in memory. Templates are typically
// small (one asset's worth of results); the load-test workload reuses them
// across many synthetic assets, so paying the I/O cost once at startup keeps
// the hot path allocation-light.
//
// Returns an error if dir contains no usable scan databases. A scan database
// missing its embedded Asset proto (older schema 1.0 files) is rejected with a
// clear error message — there is no way to call SynchronizeAssets without it.
func LoadTemplates(ctx context.Context, dir string) ([]*Template, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read templates directory %q", dir)
	}

	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	sort.Strings(paths)

	if len(paths) == 0 {
		return nil, fmt.Errorf("no .db files found in %q", dir)
	}

	templates := make([]*Template, 0, len(paths))
	for _, p := range paths {
		t, err := loadTemplate(ctx, p)
		if err != nil {
			return nil, errors.Wrapf(err, "template %s", p)
		}
		templates = append(templates, t)
	}
	return templates, nil
}

func loadTemplate(ctx context.Context, path string) (*Template, error) {
	store, err := scandb.NewSqliteScanDataStoreReader(path)
	if err != nil {
		return nil, errors.Wrap(err, "open scan db")
	}
	defer store.Close()

	asset, err := store.GetAsset(ctx)
	if err != nil {
		if errors.Is(err, policy.ErrAssetNotFound) {
			return nil, fmt.Errorf("scan db has no embedded asset (schema 1.0 file?); regenerate it with a cnspec build that writes assets")
		}
		return nil, errors.Wrap(err, "read asset")
	}

	codeIDs, err := store.GetAssetFilters(ctx)
	if err != nil {
		if errors.Is(err, policy.ErrAssetFiltersNotFound) {
			return nil, fmt.Errorf("scan db has no asset filters; recapture with `cnspec scan --output-scan-db`")
		}
		return nil, errors.Wrap(err, "read asset filters")
	}

	t := &Template{
		Path:          path,
		Asset:         asset,
		FilterCodeIDs: codeIDs,
		Data:          make(map[string]*llx.Result),
	}

	if err := store.StreamScores(ctx, func(s *policy.Score) error {
		t.Scores = append(t.Scores, s)
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "stream scores")
	}

	if err := store.StreamData(ctx, func(codeID string, r *llx.Result) error {
		t.Data[codeID] = r
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "stream data")
	}

	if err := store.StreamRisks(ctx, func(r *policy.ScoredRiskFactor) error {
		t.Risks = append(t.Risks, r)
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "stream risks")
	}

	if len(t.Scores) == 0 {
		return nil, fmt.Errorf("scan db has no scores")
	}
	return t, nil
}
