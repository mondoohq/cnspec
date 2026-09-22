// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mondoo.com/mql/test"
)

var (
	// cnspecBin is the absolute path of the binary under test. Absolute because
	// scenarios run from the package directory while the k8s tier shells out
	// from elsewhere, and a relative path would resolve differently.
	cnspecBin string

	// cnspecVersion is `cnspec version` for that binary, printed once at start.
	// A failing run has to name the artifact it exercised: "cnspec" is not an
	// answer when the whole point of the suite is to tell a release candidate
	// apart from a local build.
	cnspecVersion string

	// absentConfig points at a file that is deliberately never created. See
	// childEnv.
	absentConfig string

	// artifactDir collects the reports of failing scenarios, for CI upload.
	artifactDir string
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "cnspec-integration-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration setup:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	absentConfig = filepath.Join(dir, "absent-mondoo.yml")

	artifactDir, err = filepath.Abs("artifacts")
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration setup:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "integration setup:", err)
		os.Exit(1)
	}

	if err := resolveBinary(); err != nil {
		fmt.Fprintln(os.Stderr, "integration setup:", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// resolveBinary picks the cnspec the suite exercises: the artifact named by
// CNSPEC_BINARY, or a fresh build of this working tree.
//
// The artifact path is what lets this suite gate a release. Packaging, ldflags
// and the production build tag are part of what ships, so a suite that can only
// build the working tree cannot answer the question a pre-release run is asked:
// does *this* artifact work.
func resolveBinary() error {
	if p := strings.TrimSpace(os.Getenv("CNSPEC_BINARY")); p != "" {
		abs, err := filepath.Abs(p)
		if err != nil {
			return fmt.Errorf("CNSPEC_BINARY=%q: %w", p, err)
		}
		fi, err := os.Stat(abs)
		if err != nil {
			return fmt.Errorf("CNSPEC_BINARY=%q: %w", p, err)
		}
		if fi.IsDir() || fi.Mode()&0o111 == 0 {
			return fmt.Errorf("CNSPEC_BINARY=%q is not an executable file", abs)
		}
		cnspecBin = abs
	} else {
		abs, err := filepath.Abs("cnspec")
		if err != nil {
			return err
		}
		// The package directory, not apps/cnspec/cnspec.go: only a package path
		// links rsrc_windows_*.syso, and it is what `make cnspec/build/windows`
		// uses. -tags production matches LDFLAGSDIST, so the binary carries the
		// same build constraints as a release.
		build := exec.Command("go", "build", "-tags", "production", "-o", abs, "../../apps/cnspec")
		// BuildEnv strips GOCOVERDIR. Without it, running this package under
		// -cover leaves GOCOVERDIR set in the child, and a binary not built
		// with -cover fails at exit trying to write coverage data.
		build.Env = test.BuildEnv()
		if out, err := build.CombinedOutput(); err != nil {
			return fmt.Errorf("building cnspec: %w\n%s", err, out)
		}
		cnspecBin = abs
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cnspecBin, "version")
	cmd.Env = childEnv()
	// Output, not CombinedOutput: cnspec logs to stderr, so the provider
	// shorthand warnings it emits on startup would end up in the version
	// string. Stderr is still captured, for the failure message only.
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		// Also the arch smoke test: a linux/amd64 tarball unpacked on an arm64
		// runner fails here, with one clear line instead of inside a scan.
		return fmt.Errorf("%s version: %w\n%s", cnspecBin, err, stderr.String())
	}
	cnspecVersion = strings.TrimSpace(string(out))

	fmt.Fprintf(os.Stderr, "integration: binary  %s\n", cnspecBin)
	fmt.Fprintf(os.Stderr, "integration: version %s\n", cnspecVersion)
	fmt.Fprintf(os.Stderr, "integration: providers %s\n", providersPath())
	return nil
}

func providersPath() string {
	if p := os.Getenv("PROVIDERS_PATH"); p != "" {
		return p
	}
	return "(default: $HOME/.config/mondoo/providers)"
}

// childEnv builds the environment every cnspec invocation runs with.
//
// Every MONDOO_* variable is dropped and MONDOO_CONFIG_PATH is pointed at a
// file that does not exist. This is not hygiene, it is correctness: mql's
// config loader reads MONDOO_CONFIG_BASE64, then $MONDOO_CONFIG_PATH, then
// ~/.config/mondoo/mondoo.yml, and binds the rest under viper's "mondoo"
// prefix. On a developer machine with a service account the scan authenticates
// against the platform and resolves policies upstream; worse, an inventory.yml
// sitting next to that config is auto-discovered and silently retargets the
// scan at something else entirely. Both were observed while building this
// suite: the first scan attempted here failed with "could not initialize client
// authentication" because it had picked up a local inventory, having never been
// asked to.
//
// PROVIDERS_PATH, KUBECONFIG, DOCKER_HOST and HOME are inherited unchanged --
// they are how the caller chooses which providers and which targets are used.
func childEnv(extra ...string) []string {
	base := test.BuildEnv() // strips GOCOVERDIR
	env := make([]string, 0, len(base)+len(extra)+1)
	for _, e := range base {
		if strings.HasPrefix(e, "MONDOO_") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "MONDOO_CONFIG_PATH="+absentConfig)
	return append(env, extra...)
}
