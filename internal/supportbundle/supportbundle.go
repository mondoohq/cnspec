// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package supportbundle gathers the artifacts a cnspec scan produces into a
// single directory that users can hand to Mondoo support for analysis.
// Activated via --collect-support-bundle on `cnspec scan`.
//
// Layout produced:
//
//	<bundle>/
//	  manifest.json           # cnspec/cnquery versions, OS/arch, scan args, host
//	  providers.json          # installed providers + versions
//	  debug.log               # zerolog tee with RFC3339Nano timestamps
//	  debug/                  # the scandump.Run directory
//	    mondoo-debug-inventory-unresolved.json   # cnquery side (via logger.DumpLocal)
//	    report.json                              # run-level cnspec
//	    assets-resolved.json
//	    resolved_mql_bundle.mql.yaml
//	    <asset>/                                  # one directory per scanned asset
//	      assetBundle.yaml
//	      policyFilters.yaml
//	      assetFilters.yaml
//	      resolvedPolicy.json
//	      resolved-policy.dot
//	      filter-queries.dot
package supportbundle

import (
	"context"
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
	"go.mondoo.com/cnspec/v13/internal/scandump"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/providers"
)

// debugSubdir is the directory under the bundle root that holds per-run /
// per-asset debug artifacts. It's the path cnquery-side dumps are pointed at
// via logger.DumpLocal, and is where scandump.Run is rooted.
const debugSubdir = "debug"

// Bundle owns a support-bundle output directory. Activate stands up the dump
// pipeline and global-state mutations; Finalize writes the metadata files
// and restores everything Activate touched.
type Bundle struct {
	// Dir is the absolute bundle root.
	Dir string
	// Args are the command-line arguments recorded in the manifest.
	Args []string

	started    time.Time
	debugDir   string
	logFile    *os.File
	prevDump   string
	prevLogger zerolog.Logger
	prevLevel  zerolog.Level
	run        *scandump.Run
	finalizeMu sync.Mutex
	finalized  bool
	announced  bool
}

// New creates a bundle directory at the given path. If path is empty, a
// timestamped directory under the current working directory is used.
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

	return &Bundle{
		Dir:      abs,
		started:  started,
		debugDir: filepath.Join(abs, debugSubdir),
	}, nil
}

// Activate stands up the dump pipeline:
//
//   - forces zerolog to debug level so dump helpers fire,
//   - creates a scandump.Run under <bundle>/debug and attaches it to ctx,
//   - points cnquery's logger.DumpLocal at the same dir so its
//     inventory-unresolved dump lands in the bundle,
//   - tees zerolog output to <bundle>/debug.log with RFC3339Nano timestamps.
//
// Returns the augmented context. Must be paired with Finalize.
func (b *Bundle) Activate(parent context.Context) (context.Context, error) {
	// Force debug level so dump helpers and cnquery's gated DebugDumpJSON
	// actually fire.
	b.prevLevel = zerolog.GlobalLevel()
	if b.prevLevel > zerolog.DebugLevel {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	run, err := scandump.NewRun(b.debugDir)
	if err != nil {
		return parent, errors.Wrap(err, "failed to create scandump run")
	}
	b.run = run

	// Point cnquery's mql/logger.DebugDumpJSON at the same directory. It
	// writes "<DumpLocal><name>.json", so the trailing prefix is intentional.
	b.prevDump = logger.DumpLocal
	logger.DumpLocal = filepath.Join(run.Dir, "mondoo-debug-")

	// Tee logs to debug.log with full timestamps; the CLI console writer
	// strips them.
	logPath := filepath.Join(b.Dir, "debug.log")
	f, err := os.Create(logPath)
	if err != nil {
		return parent, errors.Wrap(err, "failed to create support-bundle debug log")
	}
	b.logFile = f

	consoleSink := zerolog.ConsoleWriter{
		Out:        logger.LogOutputWriter,
		NoColor:    false,
		TimeFormat: time.Kitchen,
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
	return scandump.WithRun(parent, run), nil
}

// HookFatal attaches a zerolog hook that finalizes the bundle when a Fatal
// event fires. zerolog runs hooks before its ExitFunc, so this is our only
// chance to flush when callers reach log.Fatal().
func (b *Bundle) HookFatal() {
	log.Logger = log.Logger.Hook(zerolog.HookFunc(func(_ *zerolog.Event, level zerolog.Level, _ string) {
		if level == zerolog.FatalLevel {
			_ = b.Finalize()
			fmt.Fprintf(os.Stderr, "support bundle written to: %s\n", b.Dir)
		}
	}))
}

// FinalizeAndAnnounce wraps Finalize with a user-visible note on the bundle
// path. Safe to call multiple times; only announces once.
func (b *Bundle) FinalizeAndAnnounce(w io.Writer) {
	if b == nil {
		return
	}
	if err := b.Finalize(); err != nil {
		log.Warn().Err(err).Msg("support bundle finalize had errors")
	}
	if !b.announced {
		fmt.Fprintf(w, "support bundle written to: %s\n", b.Dir)
		b.announced = true
	}
}

// Finalize writes manifest.json + providers.json, restores global logger
// state, and closes the debug log. Safe to call more than once.
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
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return errors.New("support bundle finalize encountered errors: " + strings.Join(msgs, "; "))
	}
	return nil
}

// Manifest is the metadata blob written to manifest.json.
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
	host, _ := os.Hostname()

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

// loggedEnvVars is the curated list of environment variables we record in
// the manifest. We deliberately do NOT dump os.Environ() — it may contain
// credentials.
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

// ProvidersDoc is the on-disk shape of providers.json.
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
			Builtin:    !p.HasBinary,
		})
	}

	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(b.Dir, "providers.json"), raw, 0o644)
}
