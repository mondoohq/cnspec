// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	subject "go.mondoo.com/cnspec/v11/internal/onboarding"
)

func TestWriteHCL(t *testing.T) {
	t.Run("without location", func(t *testing.T) {
		location, err := subject.WriteHCL("code", "", "test")
		if assert.Nil(t, err) {
			defer os.RemoveAll(location)
		}
		assert.Contains(t, location, ".config/mondoo/onboarding/test")
	})

	t.Run("with a custom location", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "unit-test")
		assert.Nil(t, err)
		location, err := subject.WriteHCL("code", dir, "test")
		assert.Nil(t, err)
		assert.Equal(t, dir, location)
		assert.FileExists(t, filepath.Join(dir, "main.tf"))
	})
}
