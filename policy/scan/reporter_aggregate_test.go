// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnspec/v10/policy"
)

func TestAggregateReport(t *testing.T) {

	b := &policy.Bundle{
		Policies: []*policy.Policy{
			{
				Uid:  "policy1",
				Name: "Policy 1",
			},
		},
	}

	r := NewAggregateReporter()
	r.AddBundle(b)
	assert.Equal(t, r.bundle, b)

	b2 := &policy.Bundle{
		Policies: []*policy.Policy{
			{
				Uid:  "policy2",
				Name: "Policy 2",
			},
		},
	}

	r.AddBundle(b2)
	assert.Equal(t, r.bundle, policy.Merge(b, b2))
}
