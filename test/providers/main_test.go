// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain runs the suite against no Mondoo configuration at all.
//
// The cnspec binary these tests build reads MONDOO_CONFIG_BASE64, then
// $MONDOO_CONFIG_PATH, then ~/.config/mondoo/mondoo.yml, and auto-discovers an
// inventory.yml sitting next to whichever config it found. On a machine with a
// service account configured, every scan here therefore authenticates against
// the platform, and an inventory next to the config silently retargets it at
// whatever that inventory names. The assertions are written for an incognito
// scan of the target on the command line, so the suite fails on exactly the
// machines of the people most likely to run it -- while passing in CI, where no
// config exists.
//
// Every MONDOO_* variable is removed and MONDOO_CONFIG_PATH is pointed at a file
// that is never created, so the config loader finds nothing and the binary runs
// incognito. The mutation is process-wide on purpose: the CLI runner builds each
// child's environment from os.Environ(). Tests that need a MONDOO_* variable set
// it themselves inside the test body, after this has run.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "cnspec-providers-test-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "test setup:", err)
		os.Exit(1)
	}

	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, "MONDOO_") {
			os.Unsetenv(name) //nolint:errcheck
		}
	}
	os.Setenv("MONDOO_CONFIG_PATH", filepath.Join(dir, "absent-mondoo.yml")) //nolint:errcheck
	// The binary under test is a source build with no release version, which
	// resolves providers from the stable channel. Preview holds the providers
	// this branch is released with.
	os.Setenv("MONDOO_UPDATE_CHANNEL", "preview") //nolint:errcheck

	code := m.Run()
	os.RemoveAll(dir) //nolint:errcheck
	os.Exit(code)
}
