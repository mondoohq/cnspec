// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPolicyMrn(t *testing.T) {
	// given
	namespace := "test-namespace"
	uid := "test-uid"

	// when
	mrn := NewPolicyMrn(namespace, uid)

	// then
	assert.Equal(t, "//registry.mondoo.com/namespace/test-namespace/policies/test-uid", mrn)
}
