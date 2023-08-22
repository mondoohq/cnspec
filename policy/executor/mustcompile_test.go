// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMustCompile(t *testing.T) {
	q := MustCompile("mondoo.version")
	assert.NotNil(t, q)

	checksum := MustGetOneDatapoint(MustCompile("mondoo.version"))
	assert.Equal(t, "J4anmJ+mXJX380Qslh563U7Bs5d6fiD2ghVxV9knAU0iy/P+IVNZsDhBbCmbpJch3Tm0NliAMiaY47lmw887Jw==", checksum)
}
