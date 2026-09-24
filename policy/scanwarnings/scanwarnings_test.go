// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scanwarnings

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDedupeAndCap(t *testing.T) {
	t.Run("empty input returns nil", func(t *testing.T) {
		assert.Nil(t, DedupeAndCap(nil))
	})

	t.Run("nil errors are skipped", func(t *testing.T) {
		out := DedupeAndCap([]error{nil, nil})
		assert.Empty(t, out)
	})

	t.Run("duplicate messages collapse to one", func(t *testing.T) {
		errs := []error{
			errors.New("the 'os' provider crashed: connection refused"),
			errors.New("the 'os' provider crashed: connection refused"),
			errors.New("the 'aws' provider crashed: EOF"),
		}
		out := DedupeAndCap(errs)
		assert.ElementsMatch(t, []string{
			"the 'os' provider crashed: connection refused",
			"the 'aws' provider crashed: EOF",
		}, out)
	})

	t.Run("count is capped", func(t *testing.T) {
		errs := make([]error, 0, Max+10)
		for i := 0; i < Max+10; i++ {
			errs = append(errs, fmt.Errorf("distinct crash #%d", i))
		}
		out := DedupeAndCap(errs)
		assert.Len(t, out, Max)
	})

	t.Run("message length is capped", func(t *testing.T) {
		long := strings.Repeat("x", MaxLen+500)
		out := DedupeAndCap([]error{errors.New(long)})
		require.Len(t, out, 1)
		assert.Len(t, out[0], MaxLen)
	})
}
