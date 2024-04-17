// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package examples

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
	"go.mondoo.com/cnspec/v11/internal/bundle"
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
		"go.mondoo.com/cnquery/v9/providers/os",
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
	result, err := bundle.Lint(mock.Schema(), files...)
	require.NoError(t, err)
	assert.False(t, result.HasError())
}
