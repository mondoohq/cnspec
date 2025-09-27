// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cli

import (
	"os"
	"sync"
	"testing"

	cmdtest "github.com/google/go-cmdtest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v12/apps/cnspec/cmd"
)

var (
	once      sync.Once
	cnspecCmd *cobra.Command
)

func setup() {
	var err error
	cnspecCmd, err = cmd.BuildRootCmd()
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	ret := m.Run()
	os.Exit(ret)
}

func TestCompare(t *testing.T) {
	t.Skip("temporary disable test until we know why the PROVIDERS_PATH env var is not ignoring system providers")
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
	tmpDirs := []string{}
	ts.Setup = func(t string) error {
		// Use a different providers dir for every test
		providersDir, err := os.MkdirTemp(os.TempDir(), "providers-*")
		if err != nil {
			return err
		}
		tmpDirs = append(tmpDirs, providersDir)
		return os.Setenv("PROVIDERS_PATH", providersDir)
	}
	ts.Run(t, false) // set to true to update test files
	for _, dir := range tmpDirs {
		err := os.RemoveAll(dir)
		assert.NoError(t, err, "failed to remove temporary providers dir %s", dir)
	}
}
