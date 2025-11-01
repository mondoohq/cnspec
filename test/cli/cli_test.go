// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cli

import (
	"os"
	"sync"
	"testing"

	cmdtest "github.com/google/go-cmdtest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers"
	"go.mondoo.com/cnspec/v12/apps/cnspec/cmd"
)

var (
	once         sync.Once
	cnspecCmd    *cobra.Command
	providersDir string
)

func setup() {
	var err error

	providersDir = ".providers"

	// we need to set the providers path to the temporary directory explicitly since the init function is called
	// before the tests are run, therefore os.Setenv("PROVIDERS_PATH", providersDir) would be too late
	providers.CustomProviderPath = providersDir
	providers.DefaultPath = providersDir
	providers.CachedProviders = nil

	cnspecCmd, err = cmd.BuildRootCmd()
	if err != nil {
		panic(err)
	}

}

func TestMain(m *testing.M) {
	// remove the temporary providers directory after all tests have run
	defer func() {
		_ = os.RemoveAll(providersDir)
	}()

	ret := m.Run()
	os.Exit(ret)
}

func TestCompare(t *testing.T) {
	once.Do(setup)

	// NOTE: those tests do not expect any provider to be available
	ts, err := cmdtest.Read("testdata")
	require.NoError(t, err)

	ts.DisableLogging = true
	ts.Commands["cnspec"] = cmdtest.InProcessProgram("cnspec", func() int {
		err := cnspecCmd.Execute()
		if err != nil {
			return 1
		}
		return 0
	})
	ts.Setup = func(t string) error {
		// clean all files stored in the providers path
		err := os.RemoveAll(providersDir)
		if err != nil {
			return err
		}
		err = os.MkdirAll(providersDir, 0o755)
		if err != nil {
			return err
		}
		// no other setup required
		return nil
	}
	ts.Run(t, false) // set to true to update test files
}
