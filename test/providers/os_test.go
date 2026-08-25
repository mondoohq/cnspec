// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/test"
)

var once sync.Once

const mqlPackagesQuery = "packages"

type mqlPackages []struct {
	Packages []struct {
		Name    string `json:"name,omitempty"`
		Version string `json:"version,omitempty"`
	} `json:"packages.list,omitempty"`
}

const mqlPlatformQuery = "asset.platform"

type mqlPlatform []struct {
	Platform string `json:"asset.platform,omitempty"`
}

type connections []struct {
	name   string
	binary string
	args   []string
	tests  []mqlTest
}

type mqlTest struct {
	query    string
	expected func(*testing.T, test.Runner)
}

// decodeJSON unmarshals the runner's stdout into dst.
//
// It fails the subtest immediately rather than merely recording an assertion,
// because every caller goes on to index into the decoded result. Continuing
// past a decode error would reach that index on an empty slice and panic,
// which takes down the whole test binary instead of this one subtest. The
// captured output is included so a CI failure says why the payload was
// unusable, not just that it was.
func decodeJSON(t *testing.T, r test.Runner, dst any) {
	t.Helper()
	if err := r.Json(dst); err != nil {
		t.Fatalf("could not decode JSON output: %v\n--- stdout ---\n%s\n--- stderr ---\n%s",
			err, r.Stdout(), r.Stderr())
	}
}

// requireResults fails the subtest when a query returned no rows.
//
// A well-formed but empty JSON array decodes without error, so the length has
// to be checked separately before indexing. This is the case a flaky container
// pull or an unreachable target produces.
func requireResults(t *testing.T, r test.Runner, n int) {
	t.Helper()
	if n == 0 {
		t.Fatalf("query returned no results\n--- stdout ---\n%s\n--- stderr ---\n%s",
			r.Stdout(), r.Stderr())
	}
}

func TestOsProviderSharedTests(t *testing.T) {
	once.Do(setup)

	connections := connections{
		{
			name:   "local",
			binary: "./cnspec",
			args:   []string{"run", "local"},
			tests: []mqlTest{
				{
					mqlPackagesQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPackages
						decodeJSON(t, r, &c)
						requireResults(t, r, len(c))

						x := c[0]
						assert.NotNil(t, x.Packages)
						assert.True(t, len(x.Packages) > 0)
					},
				},
				{
					mqlPlatformQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPlatform
						decodeJSON(t, r, &c)
						requireResults(t, r, len(c))

						x := c[0]
						assert.True(t, len(x.Platform) > 0)
					},
				},
			},
		},
		{
			name:   "fs",
			binary: "./cnspec",
			args:   []string{"run", "fs", "--path", "./testdata/fs"},
			tests: []mqlTest{
				{
					mqlPackagesQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPackages
						decodeJSON(t, r, &c)
						requireResults(t, r, len(c))

						x := c[0]
						assert.NotNil(t, x.Packages)
						assert.True(t, len(x.Packages) > 0)
					},
				},
				{
					mqlPlatformQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPlatform
						decodeJSON(t, r, &c)
						requireResults(t, r, len(c))

						x := c[0]
						assert.Equal(t, "debian", x.Platform)
					},
				},
			},
		},
		{
			name:   "docker",
			binary: "./cnspec",
			args:   []string{"run", "docker", "alpine:latest"},
			tests: []mqlTest{
				{
					mqlPackagesQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPackages
						decodeJSON(t, r, &c)
						requireResults(t, r, len(c))

						x := c[0]
						assert.NotNil(t, x.Packages)
						assert.True(t, len(x.Packages) > 0)
					},
				},
				{
					mqlPlatformQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPlatform
						decodeJSON(t, r, &c)
						requireResults(t, r, len(c))

						x := c[0]
						assert.Equal(t, "alpine", x.Platform)
					},
				},
			},
		},
	}

	// iterate over all tests for all connections
	for _, cc := range connections {
		for _, tt := range cc.tests {
			t.Run(cc.name+"/"+tt.query, func(t *testing.T) {
				r := test.NewCliTestRunner(cc.binary, append(cc.args, "-c", tt.query, "-j")...)
				err := r.Run()
				require.NoError(t, err)
				assert.Equal(t, 0, r.ExitCode())
				assert.NotNil(t, r.Stdout())
				assert.NotNil(t, r.Stderr())

				tt.expected(t, r)
			})
		}
	}
}

func TestProvidersEnvVarsLoading(t *testing.T) {
	once.Do(setup)

	t.Run("command WITHOUT path should report the missing path", func(t *testing.T) {
		r := test.NewCliTestRunner("./cnspec", "run", "fs", "-c", mqlPackagesQuery, "-j")
		err := r.Run()
		require.NoError(t, err)

		// Without MONDOO_PATH (and without --path) the fs asset cannot connect.
		// That is a misconfiguration, so it exits non-zero rather than
		// reporting an empty package list, which would read as "this
		// filesystem has no packages installed". Matches the behaviour
		// pinned by mondoohq/mql#10267.
		assert.Equal(t, 1, r.ExitCode())
		assert.NotNil(t, r.Stderr())
		assert.Contains(t, string(r.Stderr()), "missing filesystem mount path")
	})
	t.Run("command WITH path should find packages", func(t *testing.T) {
		os.Setenv("MONDOO_PATH", "./testdata/fs")
		defer os.Unsetenv("MONDOO_PATH")
		// Note we are not passing the flag "--path ./testdata/fs"
		r := test.NewCliTestRunner("./cnspec", "run", "fs", "-c", mqlPackagesQuery, "-j")
		err := r.Run()
		require.NoError(t, err)
		assert.Equal(t, 0, r.ExitCode())
		assert.NotNil(t, r.Stdout())
		assert.NotNil(t, r.Stderr())

		var c mqlPackages
		err = r.Json(&c)
		assert.NoError(t, err)

		// Should have packages
		if assert.NotEmpty(t, c) {
			x := c[0]
			assert.NotNil(t, x.Packages)
			assert.True(t, len(x.Packages) > 0)
		}
	})

	t.Run("command with flags set to not bind to config (ConfigEntry=\"-\")", func(t *testing.T) {
		t.Run("should work via direct flag", func(t *testing.T) {
			r := test.NewCliTestRunner("./cnspec", "run", "ssh", "localhost", "-c", "ls", "-p", "test", "-v")
			err := r.Run()
			require.NoError(t, err)
			// The subject here is flag binding, which is observable in stderr.
			// Whether ssh://localhost actually connects depends on the machine
			// running the test, so the exit code is deliberately not asserted.
			assert.NotNil(t, r.Stdout())
			if assert.NotNil(t, r.Stderr()) {
				assert.Contains(t, string(r.Stderr()), "skipping config binding for password")
				assert.Contains(t, string(r.Stderr()), "enabled ssh password authentication")
			}
		})
		t.Run("should NOT work via config/env-vars", func(t *testing.T) {
			os.Setenv("MONDOO_PASSWORD", "test")
			defer os.Unsetenv("MONDOO_PASSWORD")
			r := test.NewCliTestRunner("./cnspec", "run", "ssh", "localhost", "-c", "ls", "-v")
			err := r.Run()
			require.NoError(t, err)
			// As above: the assertion is about config binding, not reachability.
			assert.NotNil(t, r.Stdout())
			if assert.NotNil(t, r.Stderr()) {
				assert.Contains(t, string(r.Stderr()), "skipping config binding for password")
				assert.NotContains(t, string(r.Stderr()), "enabled ssh password authentication")
			}
		})
	})
}
