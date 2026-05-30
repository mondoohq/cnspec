// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package scandump captures the per-scan debug artifacts (asset bundle,
// inventory, resolved policy, report, graph .dot files, …) into a single
// directory tree organized per asset. A scan can target many assets in one
// invocation; each asset gets its own subdirectory so files from concurrent
// or sequential assets don't overwrite each other.
//
// The target directory is plumbed through context.Context. Call WithRun at
// the entry point of the command, then WithAsset before per-asset work
// begins. JSON, YAML, and Create are no-ops when no Run is attached.
package scandump

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"sigs.k8s.io/yaml"
)

// Run owns a per-scan-invocation output directory. The Dir field is what
// callers want when constructing paths for things scandump doesn't write
// itself (e.g. the support bundle's debug.log).
type Run struct {
	Dir string

	mu        sync.Mutex
	usedNames map[string]int
}

// NewRun creates a Run rooted at dir. An empty dir returns an inactive Run
// — every helper is a no-op against it — which lets callers feed in their
// "do we want dumps?" decision without scattering nil checks.
func NewRun(dir string) (*Run, error) {
	if dir == "" {
		return &Run{}, nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve scandump path")
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, errors.Wrap(err, "failed to create scandump directory")
	}
	return &Run{Dir: abs, usedNames: map[string]int{}}, nil
}

// Active reports whether the Run will write files. Useful for skipping
// expensive work (e.g. fetching the asset bundle) when dumping is off.
func (r *Run) Active() bool {
	return r != nil && r.Dir != ""
}

// AssetDir reserves a subdirectory for the given asset name. The first call
// for a name returns "<root>/<name>"; subsequent calls return "<name>-1",
// "<name>-2", …. The directory is created before returning.
//
// Names go through sanitize to keep the filesystem happy.
func (r *Run) AssetDir(name string) (string, error) {
	if !r.Active() {
		return "", nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	base := sanitize(name)
	if base == "" {
		base = "asset"
	}
	n := r.usedNames[base]
	final := base
	if n > 0 {
		final = base + "-" + strconv.Itoa(n)
	}
	r.usedNames[base] = n + 1

	dir := filepath.Join(r.Dir, final)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", errors.Wrapf(err, "failed to create asset dump directory %q", dir)
	}
	return dir, nil
}

type ctxKey struct{}

// scope is what the context carries: the Run (so children can register more
// assets) plus the current writeDir. writeDir empty means "fall back to
// Run.Dir" — useful for run-level dumps like the final report.
type scope struct {
	run      *Run
	writeDir string
}

// WithRun attaches r to ctx. Run-level dump calls (no per-asset scope) will
// write to r.Dir.
func WithRun(ctx context.Context, r *Run) context.Context {
	if !r.Active() {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, &scope{run: r})
}

// WithAsset reserves a per-asset directory and returns a ctx whose dump
// helpers write into it. If ctx has no Run attached, WithAsset is a no-op
// and returns ctx unchanged.
//
// The returned (string, error) is the asset directory path (useful for
// logging) and any mkdir failure. A nil error with empty path means dumps
// are disabled in this ctx.
func WithAsset(ctx context.Context, name string) (context.Context, string, error) {
	s, _ := ctx.Value(ctxKey{}).(*scope)
	if s == nil || !s.run.Active() {
		return ctx, "", nil
	}
	dir, err := s.run.AssetDir(name)
	if err != nil {
		return ctx, "", err
	}
	return context.WithValue(ctx, ctxKey{}, &scope{run: s.run, writeDir: dir}), dir, nil
}

// FromContext returns the active dump directory and whether dumping is on.
// Most callers should prefer JSON/YAML/Create; this is for tests and for
// callers that need to hand a path to other code (e.g. a graph writer).
func FromContext(ctx context.Context) (string, bool) {
	s, _ := ctx.Value(ctxKey{}).(*scope)
	if s == nil {
		return "", false
	}
	if s.writeDir != "" {
		return s.writeDir, true
	}
	if s.run.Active() {
		return s.run.Dir, true
	}
	return "", false
}

// Active reports whether ctx has a writable Run/Asset scope attached.
func Active(ctx context.Context) bool {
	_, ok := FromContext(ctx)
	return ok
}

// JSON writes obj as pretty JSON to <scope>/<name>.json. No-op when ctx has
// no active scope. Errors are intentionally swallowed (logged as warnings)
// — dump failures should not abort a scan.
func JSON(ctx context.Context, name string, obj any) {
	dir, ok := FromContext(ctx)
	if !ok {
		return
	}
	raw, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		warn(name, errors.Wrap(err, "marshal JSON"))
		return
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		warn(name, errors.Wrap(err, "write JSON"))
	}
}

// YAML writes obj as YAML to <scope>/<name>.yaml. No-op when ctx has no
// active scope.
func YAML(ctx context.Context, name string, obj any) {
	dir, ok := FromContext(ctx)
	if !ok {
		return
	}
	raw, err := yaml.Marshal(obj)
	if err != nil {
		warn(name, errors.Wrap(err, "marshal YAML"))
		return
	}
	path := filepath.Join(dir, name+".yaml")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		warn(name, errors.Wrap(err, "write YAML"))
	}
}

// Create opens <scope>/<filename> for writing. Returns (nil, nil) when ctx
// has no active scope — the typical caller pattern is:
//
//	f, err := scandump.Create(ctx, "graph.dot")
//	if err != nil { … }
//	if f == nil { return }
//	defer f.Close()
//	// …
//
// Caller is responsible for closing the file.
func Create(ctx context.Context, filename string) (*os.File, error) {
	dir, ok := FromContext(ctx)
	if !ok {
		return nil, nil
	}
	return os.Create(filepath.Join(dir, filename))
}

// sanitize keeps letters, digits, and "._-"; everything else becomes "_".
// We want stable, predictable directory names — asset names can contain
// slashes, colons (MRNs), spaces, etc.
func sanitize(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// warn is the central place to deal with dump failures. We don't import
// zerolog here because importing it pulls in side-effecting init code; the
// dump helpers should be safe to use from anywhere. Failures end up on
// stderr.
var warn = func(name string, err error) {
	// Best-effort logging; ignore errors writing to stderr.
	_, _ = os.Stderr.WriteString("scandump: " + name + ": " + err.Error() + "\n")
}
