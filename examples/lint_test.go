// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package examples

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnspec/v10/internal/bundle"
	"os"
	"testing"
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
	dir := ".lint-providers"
	providers.CustomProviderPath = dir
	providers.DefaultPath = dir

	err := ensureProviders([]string{
		"go.mondoo.com/cnquery/providers/os",
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

	runtime := providers.DefaultRuntime()
	result, err := bundle.Lint(runtime.Schema(), files...)
	require.NoError(t, err)
	assert.False(t, result.HasError())
}
