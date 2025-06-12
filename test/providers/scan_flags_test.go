// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/test"
	"go.mondoo.com/cnspec/v11/policy"
)

func TestScanFlags(t *testing.T) {
	once.Do(setup)

	t.Run("successful scan without flags", func(t *testing.T) {
		r := test.NewCliTestRunner("./cnspec", "scan", "docker", "alpine:latest", "--json")
		err := r.Run()
		require.NoError(t, err)
		assert.Equal(t, 0, r.ExitCode())
		assert.NotNil(t, r.Stdout())
		assert.NotNil(t, r.Stderr())

		var c policy.ReportCollection
		err = r.Json(&c)
		assert.NoError(t, err)

		// Assert must be found
		assert.NotEmpty(t, c.Assets)
	})
	t.Run("github scan WITHOUT flags", func(t *testing.T) {
		// NOTE this will fail but, it will load the flags and fail with the right message
		r := test.NewCliTestRunner("./cnspec", "scan", "github", "repo", "foo")
		err := r.Run()
		require.NoError(t, err)
		assert.Equal(t, 0, r.ExitCode())
		assert.NotNil(t, r.Stdout())
		assert.NotNil(t, r.Stderr())

		assert.Contains(t, string(r.Stderr()),
			"a valid GitHub authentication is required",
		)
	})
	t.Run("github scan WITH flags but missing app auth key", func(t *testing.T) {
		// NOTE this will fail but, it will load the flags and fail with the right message
		r := test.NewCliTestRunner("./cnspec", "scan", "github", "repo", "foo",
			"--app-id", "123", "--app-installation-id", "456",
		)
		err := r.Run()
		require.NoError(t, err)
		assert.Equal(t, 1, r.ExitCode())
		assert.NotNil(t, r.Stdout())
		assert.NotNil(t, r.Stderr())

		assert.Contains(t, string(r.Stderr()),
			"could not parse private key", // expected! it means we loaded the flags
		)
	})
	t.Run("github scan WITH all required flags for app auth", func(t *testing.T) {
		// NOTE this will fail but, it will load the flags and fail with the right message
		r := test.NewCliTestRunner("./cnspec", "scan", "github", "repo", "foo",
			"--app-id", "123", "--app-installation-id", "456", "--app-private-key", "private-key.pem",
		)
		err := r.Run()
		require.NoError(t, err)
		assert.Equal(t, 1, r.ExitCode())
		assert.NotNil(t, r.Stdout())
		assert.NotNil(t, r.Stderr())

		assert.Contains(t, string(r.Stderr()),
			"could not read private key", // expected! it means we loaded the flags
		)
	})
}
