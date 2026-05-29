// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package supportbundle gathers the artifacts a cnspec scan produces under
// DEBUG=1 (asset bundle, inventory, resolved policy, report, graph dot files,
// etc.) into a single directory that users can hand to Mondoo support for
// analysis. Activated via the --collect-support-bundle flag on `cnspec scan`.
package supportbundle

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// dumpPrefix is the prefix the upstream logger.DebugDumpJSON/YAML helpers,
// the cnspec graph executor, and the cnquery graph executor use when writing
// debug artifacts. We rely on it here to sweep any files written to CWD into
// the bundle directory at finalize time.
const dumpPrefix = "mondoo-debug-"

// Bundle owns a support-bundle output directory. Activate switches the global
// debug-dump prefix and forces debug-level logging; Finalize collects the
// final artifacts (manifest, provider versions, leftover files) and restores
// any global state we touched.
type Bundle struct {
	// Dir is the absolute path the bundle is written to.
	Dir string
	// Args are the command-line arguments recorded in the manifest.
	Args []string

	started    time.Time
	cwd        string
	logFile    *os.File
	prevDump   string
	prevLogger zerolog.Logger
	prevLevel  zerolog.Level
	finalizeMu sync.Mutex
	finalized  bool
	announced  bool
}

// New creates a bundle directory at the given path. If path is empty, a
// timestamped directory under the current working directory is used. The
// returned Bundle is not active yet; call Activate before running the scan.
func New(path string) (*Bundle, error) {
	started := time.Now().UTC()

	if path == "" {
		path = "cnspec-support-bundle-" + started.Format("20060102T150405Z")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve support-bundle path")
	}

	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, errors.Wrap(err, "failed to create support-bundle directory")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to determine working directory")
	}

	return &Bundle{
		Dir:     abs,
		started: started,
		cwd:     cwd,
	}, nil
}

// Activate redirects DebugDumpJSON/YAML output, the cnspec graph .dot file,
// and a tee'd debug log into the bundle directory, and forces debug-level
// logging if a more restrictive level was set. Must be paired with Finalize.
func (b *Bundle) Activate() error {
	// Force debug level so the dump helpers actually fire. Remember the prior
	// level so Finalize can restore it.
	b.prevLevel = zerolog.GlobalLevel()
	if b.prevLevel > zerolog.DebugLevel {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// logger.DumpLocal is a "prefix" — DebugDumpJSON writes DumpLocal+name+".json".
	// Setting it to "<dir>/mondoo-debug-" lands every existing call inside the
	// bundle dir without changing their hardcoded names.
	b.prevDump = logger.DumpLocal
	logger.DumpLocal = filepath.Join(b.Dir, dumpPrefix)

	// Tee zerolog output: keep writing to the existing console destination
	// (LogOutputWriter — the buffered stderr the CLI uses) and also write
	// to debug.log with RFC3339Nano timestamps. The CLI's compact ConsoleWriter
	// suppresses timestamps; the file writer reinstates them so support can
	// reconstruct timing.
	//
	// Note: when active, we use a plain ConsoleWriter for the console rather
	// than cnquery's custom-formatted one. That trades the cute level glyphs
	// (→, !, x) for a working fan-out. The file is what support needs anyway.
	logPath := filepath.Join(b.Dir, "debug.log")
	f, err := os.Create(logPath)
	if err != nil {
		return errors.Wrap(err, "failed to create support-bundle debug log")
	}
	b.logFile = f

	consoleSink := zerolog.ConsoleWriter{
		Out:        logger.LogOutputWriter,
		NoColor:    false,
		TimeFormat: time.Kitchen, // short timestamps for console; full ones go to file
	}
	fileSink := zerolog.ConsoleWriter{
		Out:        f,
		NoColor:    true,
		TimeFormat: time.RFC3339Nano,
	}
	b.prevLogger = log.Logger
	log.Logger = zerolog.New(zerolog.MultiLevelWriter(consoleSink, fileSink)).
		Level(zerolog.GlobalLevel()).
		With().Timestamp().Logger()

	log.Debug().Str("dir", b.Dir).Msg("support bundle collection started")
	return nil
}

// HookFatal attaches a zerolog hook that finalizes the bundle when a Fatal
// event fires. zerolog runs Done hooks before its ExitFunc, so this gives
// us a deterministic flush even when scanCmdRun calls log.Fatal().
func (b *Bundle) HookFatal() {
	log.Logger = log.Logger.Hook(zerolog.HookFunc(func(_ *zerolog.Event, level zerolog.Level, _ string) {
		if level == zerolog.FatalLevel {
			// Best-effort: errors here will be swallowed because the
			// process is about to exit anyway.
			_ = b.Finalize()
			fmt.Fprintf(os.Stderr, "support bundle written to: %s\n", b.Dir)
		}
	}))
}

// FinalizeAndAnnounce wraps Finalize with a user-visible note on the bundle
// path. Safe to call multiple times; subsequent calls are no-ops.
func (b *Bundle) FinalizeAndAnnounce(w io.Writer) {
	if b == nil {
		return
	}
	if err := b.Finalize(); err != nil {
		// Don't fail the scan over a bundle write error; just complain loudly.
		log.Warn().Err(err).Msg("support bundle finalize had errors")
	}
	if !b.announced {
		fmt.Fprintf(w, "support bundle written to: %s\n", b.Dir)
		b.announced = true
	}
}

// RecordResolvedAssets dumps the asset list from the scan report so support
// can see which assets actually got resolved/scanned. The unresolved inventory
// is captured separately by the cnquery inventory manager via DebugDumpJSON.
func (b *Bundle) RecordResolvedAssets(report *policy.ReportCollection) {
	if report == nil {
		return
	}
	// Pull the bits that identify each asset; the full report.Assets values
	// include scoring state we already dump elsewhere via "report.json".
	resolved := struct {
		Assets map[string]*inventory.Asset `json:"assets"`
		Errors map[string]string           `json:"errors,omitempty"`
	}{
		Assets: report.Assets,
		Errors: report.Errors,
	}
	raw, err := json.MarshalIndent(resolved, "", "  ")
	if err != nil {
		log.Warn().Err(err).Msg("failed to marshal resolved assets")
		return
	}
	if err := os.WriteFile(filepath.Join(b.Dir, "assets-resolved.json"), raw, 0o644); err != nil {
		log.Warn().Err(err).Msg("failed to write resolved assets")
	}
}

// Finalize writes manifest.json + providers.json, sweeps any leftover
// ./mondoo-debug-* files into the bundle dir, restores global logger state,
// and closes the debug log. Safe to call more than once.
func (b *Bundle) Finalize() error {
	b.finalizeMu.Lock()
	defer b.finalizeMu.Unlock()
	if b.finalized {
		return nil
	}
	b.finalized = true

	var errs []error

	if err := b.writeManifest(); err != nil {
		errs = append(errs, errors.Wrap(err, "manifest"))
	}
	if err := b.writeProviders(); err != nil {
		errs = append(errs, errors.Wrap(err, "providers"))
	}
	if err := b.sweepCWD(); err != nil {
		errs = append(errs, errors.Wrap(err, "sweep"))
	}

	log.Debug().Str("dir", b.Dir).Msg("support bundle collection finished")

	// Restore global state.
	logger.DumpLocal = b.prevDump
	log.Logger = b.prevLogger
	zerolog.SetGlobalLevel(b.prevLevel)

	if b.logFile != nil {
		if err := b.logFile.Close(); err != nil {
			errs = append(errs, errors.Wrap(err, "close log"))
		}
	}

	if len(errs) > 0 {
		// Return all errors joined so callers see every failure.
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return errors.New("support bundle finalize encountered errors: " + strings.Join(msgs, "; "))
	}
	return nil
}

// Manifest is the metadata blob written to manifest.json. It captures the
// versions of the binaries involved and basic host info so support can
// reproduce the environment.
type Manifest struct {
	CreatedAt  time.Time         `json:"created_at"`
	CnspecInfo string            `json:"cnspec"`
	CnquerySDK string            `json:"cnquery_sdk"`
	GoVersion  string            `json:"go_version"`
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	Hostname   string            `json:"hostname,omitempty"`
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env"`
}

func (b *Bundle) writeManifest() error {
	host, _ := os.Hostname() // best effort

	m := Manifest{
		CreatedAt:  b.started,
		CnspecInfo: cnspec.Info(),
		CnquerySDK: mql.Info(),
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Hostname:   host,
		Args:       b.Args,
		Env:        collectRelevantEnv(),
	}

	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(b.Dir, "manifest.json"), raw, 0o644)
}

// collectRelevantEnv records only the env vars that affect cnspec behavior.
// We deliberately do NOT dump os.Environ() — it may contain credentials.
var loggedEnvVars = []string{
	"DEBUG", "TRACE", "MONDOO_CONFIG_PATH", "MONDOO_CONFIG_HOME",
	"MONDOO_HOME", "MONDOO_AUTO_UPDATE", "NO_COLOR", "HTTP_PROXY",
	"HTTPS_PROXY", "NO_PROXY", "MEM_DEBUG",
}

func collectRelevantEnv() map[string]string {
	out := map[string]string{}
	for _, k := range loggedEnvVars {
		if v, ok := os.LookupEnv(k); ok {
			out[k] = v
		}
	}
	return out
}

// ProvidersDoc is the on-disk shape of providers.json. We keep the list
// alphabetized for deterministic diffs across captures.
type ProvidersDoc struct {
	Providers []ProviderEntry `json:"providers"`
}

type ProviderEntry struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	ID         string   `json:"id,omitempty"`
	Connectors []string `json:"connectors,omitempty"`
	Path       string   `json:"path,omitempty"`
	Builtin    bool     `json:"builtin"`
}

func (b *Bundle) writeProviders() error {
	all, err := providers.ListAll()
	if err != nil {
		// Don't fail the whole bundle if provider listing chokes — record what
		// we got and surface the error in the file itself.
		entry := struct {
			Error string `json:"error"`
		}{Error: err.Error()}
		raw, _ := json.MarshalIndent(entry, "", "  ")
		return os.WriteFile(filepath.Join(b.Dir, "providers.json"), raw, 0o644)
	}

	doc := ProvidersDoc{}
	for _, p := range all {
		if p == nil {
			continue
		}
		connectors := make([]string, 0, len(p.Connectors))
		for _, c := range p.Connectors {
			connectors = append(connectors, c.Name)
		}
		doc.Providers = append(doc.Providers, ProviderEntry{
			Name:       p.Name,
			Version:    p.Version,
			ID:         p.ID,
			Connectors: connectors,
			Path:       p.Path,
			Builtin:    p.HasBinary == false,
		})
	}

	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(b.Dir, "providers.json"), raw, 0o644)
}

// sweepCWD moves any leftover ./mondoo-debug-* files from the working dir
// the process started in into the bundle. This catches debug artifacts
// written by code that hardcodes the prefix without honoring DumpLocal —
// notably the cnquery-side graph executor's .dot file.
func (b *Bundle) sweepCWD() error {
	entries, err := os.ReadDir(b.cwd)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, dumpPrefix) {
			continue
		}
		src := filepath.Join(b.cwd, name)
		dst := filepath.Join(b.Dir, name)
		if src == dst {
			continue
		}
		if err := moveFile(src, dst); err != nil {
			log.Warn().Err(err).Str("file", name).Msg("failed to move debug artifact into support bundle")
		}
	}
	return nil
}

// moveFile renames src → dst when both are on the same filesystem, falling
// back to copy+remove across devices. We don't worry about partial copies on
// failure — the original stays put for the user to inspect.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}
