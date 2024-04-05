// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlastRadius(t *testing.T) {
	conf := DefaultBlastRadiusConfig
	tests := []struct {
		n         float32
		max       float32
		indicator string
	}{
		{1, 100, "s"},
		{10, 100, "m"},
		{30, 100, "l"},
		{4, 5, "s"},
		{10, 20, "m"},
		{50, 100, "l"},
	}

	for i := range tests {
		test := tests[i]
		t.Run(fmt.Sprintf("%.2f / %.2f => %s", test.n, test.max, test.indicator), func(t *testing.T) {
			assert.Equal(t, test.indicator, string(conf.Indicator(test.max, test.n)))
		})
	}
}
