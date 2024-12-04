// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/test"
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
						err := r.Json(&c)
						assert.NoError(t, err)

						x := c[0]
						assert.NotNil(t, x.Packages)
						assert.True(t, len(x.Packages) > 0)
					},
				},
				{
					mqlPlatformQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPlatform
						err := r.Json(&c)
						assert.NoError(t, err)

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
						err := r.Json(&c)
						assert.NoError(t, err)

						x := c[0]
						assert.NotNil(t, x.Packages)
						assert.True(t, len(x.Packages) > 0)
					},
				},
				{
					mqlPlatformQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPlatform
						err := r.Json(&c)
						assert.NoError(t, err)

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
						err := r.Json(&c)
						assert.NoError(t, err)

						x := c[0]
						assert.NotNil(t, x.Packages)
						assert.True(t, len(x.Packages) > 0)
					},
				},
				{
					mqlPlatformQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPlatform
						err := r.Json(&c)
						assert.NoError(t, err)

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

	t.Run("command WITHOUT path should not find any package", func(t *testing.T) {
		r := test.NewCliTestRunner("./cnspec", "run", "fs", "-c", mqlPackagesQuery, "-j")
		err := r.Run()
		require.NoError(t, err)
		assert.Equal(t, 0, r.ExitCode())
		assert.NotNil(t, r.Stdout())
		assert.NotNil(t, r.Stderr())

		var c mqlPackages
		err = r.Json(&c)
		assert.NoError(t, err)

		// No packages
		assert.Empty(t, c)
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
			assert.Equal(t, 0, r.ExitCode())
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
			assert.Equal(t, 0, r.ExitCode())
			assert.NotNil(t, r.Stdout())
			if assert.NotNil(t, r.Stderr()) {
				assert.Contains(t, string(r.Stderr()), "skipping config binding for password")
				assert.NotContains(t, string(r.Stderr()), "enabled ssh password authentication")
			}
		})
	})
}
