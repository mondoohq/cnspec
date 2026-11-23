// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy/scan"
	"go.mondoo.com/mql/cli/execruntime"
)

func TestSubmitExceptions(t *testing.T) {
	local := &execruntime.RuntimeEnv{Id: execruntime.CLIENT_ENV}
	gitlab := &execruntime.RuntimeEnv{Id: execruntime.GITLAB}
	t.Setenv("CI_DEFAULT_BRANCH", "main")

	tests := []struct {
		name   string
		mode   string
		env    *execruntime.RuntimeEnv
		branch string
		want   scan.ExceptionsSubmit
	}{
		{"auto outside CI submits nothing", "auto", local, "", scan.ExceptionsNotSubmitted},
		{"auto in CI on the default branch registers", "auto", gitlab, "main", scan.ExceptionsRegister},
		{"auto in CI on another branch only applies", "auto", gitlab, "feature", scan.ExceptionsApplyOnly},
		{"auto in CI without a branch only applies", "", gitlab, "", scan.ExceptionsApplyOnly},
		{"always registers", "always", gitlab, "feature", scan.ExceptionsRegister},
		{"always registers outside CI", "always", local, "", scan.ExceptionsRegister},
		{"never submits nothing", "never", gitlab, "main", scan.ExceptionsNotSubmitted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CI_COMMIT_BRANCH", tt.branch)
			got, err := submitExceptions(tt.mode, tt.env)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	_, err := submitExceptions("sometimes", local)
	assert.Error(t, err)
}
