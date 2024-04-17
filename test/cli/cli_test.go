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
	"go.mondoo.com/cnspec/v11/apps/cnspec/cmd"
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
	ts.Run(t, false) // set to true to update test files
}
