// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandump

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRun_EmptyPath_ReturnsInactive(t *testing.T) {
	r, err := NewRun("")
	require.NoError(t, err)
	assert.False(t, r.Active(), "empty path should produce an inactive Run that no-ops")
}

func TestNewRun_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "run")
	r, err := NewRun(dir)
	require.NoError(t, err)
	assert.True(t, r.Active())

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestAssetDir_DedupesDuplicateNames(t *testing.T) {
	r, err := NewRun(t.TempDir())
	require.NoError(t, err)

	d1, err := r.AssetDir("webserver")
	require.NoError(t, err)
	d2, err := r.AssetDir("webserver")
	require.NoError(t, err)
	d3, err := r.AssetDir("webserver")
	require.NoError(t, err)

	assert.Equal(t, "webserver", filepath.Base(d1),
		"first occurrence of a name should keep the bare name")
	assert.Equal(t, "webserver-1", filepath.Base(d2),
		"second occurrence should get -1 suffix")
	assert.Equal(t, "webserver-2", filepath.Base(d3),
		"third occurrence should get -2 suffix")
}

func TestAssetDir_DifferentNamesNoCollision(t *testing.T) {
	r, err := NewRun(t.TempDir())
	require.NoError(t, err)

	d1, err := r.AssetDir("alpha")
	require.NoError(t, err)
	d2, err := r.AssetDir("beta")
	require.NoError(t, err)
	d3, err := r.AssetDir("alpha")
	require.NoError(t, err)

	assert.Equal(t, "alpha", filepath.Base(d1))
	assert.Equal(t, "beta", filepath.Base(d2))
	assert.Equal(t, "alpha-1", filepath.Base(d3),
		"name reuse counts independently per base name")
}

func TestAssetDir_SanitizesProblematicCharacters(t *testing.T) {
	r, err := NewRun(t.TempDir())
	require.NoError(t, err)

	// Asset names can contain slashes (paths), colons (MRNs), spaces, etc.
	dir, err := r.AssetDir("//policy.api.mondoo.com/assets/abc:def 123")
	require.NoError(t, err)
	base := filepath.Base(dir)
	// No path-separating runes left.
	assert.NotContains(t, base, "/")
	assert.NotContains(t, base, ":")
	assert.NotContains(t, base, " ")
}

func TestAssetDir_EmptyNameFallsBackToAsset(t *testing.T) {
	r, err := NewRun(t.TempDir())
	require.NoError(t, err)

	dir, err := r.AssetDir("")
	require.NoError(t, err)
	assert.Equal(t, "asset", filepath.Base(dir),
		"empty asset name should fall back to a generic placeholder")
}

func TestAssetDir_InactiveRunReturnsEmpty(t *testing.T) {
	r, err := NewRun("")
	require.NoError(t, err)

	dir, err := r.AssetDir("anything")
	require.NoError(t, err)
	assert.Empty(t, dir, "inactive Run must not pretend to allocate directories")
}

func TestContext_NoRun_DumpsAreNoOps(t *testing.T) {
	ctx := context.Background()
	// Should not panic, should not write anywhere.
	JSON(ctx, "anything", map[string]string{"x": "y"})
	YAML(ctx, "anything", map[string]string{"x": "y"})
	f, err := Create(ctx, "anything.dot")
	require.NoError(t, err)
	assert.Nil(t, f, "Create with no scope should return (nil, nil)")
	assert.False(t, Active(ctx))
}

func TestContext_RunOnly_WritesAtRunRoot(t *testing.T) {
	dir := t.TempDir()
	r, err := NewRun(dir)
	require.NoError(t, err)
	ctx := WithRun(context.Background(), r)

	JSON(ctx, "manifest", map[string]string{"hello": "world"})

	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	require.NoError(t, err)
	var got map[string]string
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "world", got["hello"])
}

func TestContext_PerAsset_WritesUnderAssetDir(t *testing.T) {
	root := t.TempDir()
	r, err := NewRun(root)
	require.NoError(t, err)

	ctx := WithRun(context.Background(), r)
	ctx, assetDir, err := WithAsset(ctx, "webserver")
	require.NoError(t, err)
	require.NotEmpty(t, assetDir)

	JSON(ctx, "report", map[string]int{"score": 80})

	raw, err := os.ReadFile(filepath.Join(assetDir, "report.json"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"score": 80`)

	// And nothing was written at the run root.
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	for _, e := range entries {
		assert.True(t, e.IsDir(), "run root should only contain asset dirs, found file %q", e.Name())
	}
}

func TestContext_MultipleAssets_Isolated(t *testing.T) {
	root := t.TempDir()
	r, err := NewRun(root)
	require.NoError(t, err)
	parent := WithRun(context.Background(), r)

	ctxA, _, err := WithAsset(parent, "alpha")
	require.NoError(t, err)
	ctxB, _, err := WithAsset(parent, "beta")
	require.NoError(t, err)

	JSON(ctxA, "report", map[string]string{"who": "alpha"})
	JSON(ctxB, "report", map[string]string{"who": "beta"})

	rawA, err := os.ReadFile(filepath.Join(root, "alpha", "report.json"))
	require.NoError(t, err)
	rawB, err := os.ReadFile(filepath.Join(root, "beta", "report.json"))
	require.NoError(t, err)
	assert.Contains(t, string(rawA), "alpha")
	assert.Contains(t, string(rawB), "beta")
}

func TestContext_DuplicateAssetNames_Isolated(t *testing.T) {
	root := t.TempDir()
	r, err := NewRun(root)
	require.NoError(t, err)
	parent := WithRun(context.Background(), r)

	ctx1, dir1, err := WithAsset(parent, "node")
	require.NoError(t, err)
	ctx2, dir2, err := WithAsset(parent, "node")
	require.NoError(t, err)

	assert.NotEqual(t, dir1, dir2)
	JSON(ctx1, "marker", map[string]int{"i": 1})
	JSON(ctx2, "marker", map[string]int{"i": 2})

	raw1, err := os.ReadFile(filepath.Join(dir1, "marker.json"))
	require.NoError(t, err)
	raw2, err := os.ReadFile(filepath.Join(dir2, "marker.json"))
	require.NoError(t, err)
	assert.Contains(t, string(raw1), `"i": 1`)
	assert.Contains(t, string(raw2), `"i": 2`)
}

func TestCreate_WritesAtScope(t *testing.T) {
	root := t.TempDir()
	r, err := NewRun(root)
	require.NoError(t, err)
	ctx := WithRun(context.Background(), r)
	ctx, assetDir, err := WithAsset(ctx, "h1")
	require.NoError(t, err)

	f, err := Create(ctx, "graph.dot")
	require.NoError(t, err)
	require.NotNil(t, f)
	_, err = f.WriteString("digraph {}")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	raw, err := os.ReadFile(filepath.Join(assetDir, "graph.dot"))
	require.NoError(t, err)
	assert.Equal(t, "digraph {}", string(raw))
}

func TestActive_TrueWhenScoped(t *testing.T) {
	r, err := NewRun(t.TempDir())
	require.NoError(t, err)
	ctx := WithRun(context.Background(), r)
	assert.True(t, Active(ctx))

	assert.False(t, Active(context.Background()))
}

func TestWithAsset_InactiveRunIsNoOp(t *testing.T) {
	r, err := NewRun("")
	require.NoError(t, err)
	ctx := WithRun(context.Background(), r)

	out, dir, err := WithAsset(ctx, "anything")
	require.NoError(t, err)
	assert.Empty(t, dir)
	assert.False(t, Active(out))
}
