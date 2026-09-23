// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package examples

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/bundle"
	"go.mondoo.com/mql/cli/config"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
)

func ensureProviders(ids []string) error {
	for _, id := range ids {
		_, err := providers.EnsureProvider(providers.ProviderLookup{ID: id}, true, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestMain(m *testing.M) {
	// Resolve providers from the preview channel. This test binary carries no
	// release version, so without it the registry would pick the stable
	// channel and validate this branch's content against the previous major's
	// providers, where a field the new major removed still exists.
	viper.Set(config.KeyUpdateChannel, config.ChannelPreview)

	dir := ".lint-providers"
	providers.CustomProviderPath = dir
	providers.DefaultPath = dir

	err := ensureProviders([]string{
		// v14 dropped the version from provider IDs (mql #10268). The registry
		// only serves the versionless ID now, so looking up the v9 one here
		// fails to install anything and the suite panics before it runs.
		"go.mondoo.com/mql/providers/os",
	})
	if err != nil {
		panic(err)
	}

	exitVal := m.Run()

	// cleanup custom provider path to ensure no leftovers and other tests are not affected
	err = os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}

	os.Exit(exitVal)
}

func TestExampleLint(t *testing.T) {
	files := []string{
		"./complex.mql.yaml",
		"./example.mql.yaml",
		"./props.mql.yaml",
	}

	mock := testutils.LinuxMock()
	result, err := bundle.Lint(mock.Schema(), bundle.LintOptions{SkipProviderDownload: true}, files...)
	require.NoError(t, err)
	assert.False(t, result.HasError())
}
