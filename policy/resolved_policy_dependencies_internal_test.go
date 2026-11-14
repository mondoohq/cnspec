// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsContentMrn(t *testing.T) {
	tests := []struct {
		name string
		mrn  string
		want bool
	}{
		{"policy", "//policy.api.mondoo.app/policies/mondoo-linux-security", true},
		{"space-owned policy", "//policy.api.mondoo.app/spaces/s1/policies/custom", true},
		{"framework", "//policy.api.mondoo.app/frameworks/cis-controls-8", true},
		{"space-owned framework", "//policy.api.mondoo.app/spaces/s1/frameworks/custom", true},
		{"space", "//captain.api.mondoo.app/spaces/s1", false},
		{"organization", "//captain.api.mondoo.app/organizations/o1", false},
		{"workspace", "//captain.api.mondoo.app/spaces/s1/workspaces/w1", false},
		{"asset", "//assets.api.mondoo.app/spaces/s1/assets/a1", false},
		{"platform", "//platform.api.mondoo.app", false},
		{"query", "//policy.api.mondoo.app/queries/mondoo-linux-ssh", false},
		{"bare name", "asset1", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isContentMrn(tt.mrn))
		})
	}
}
