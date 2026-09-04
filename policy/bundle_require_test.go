// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequiredProviderNames(t *testing.T) {
	t.Run("no requirements", func(t *testing.T) {
		assert.Empty(t, RequiredProviderNames(nil))
	})

	t.Run("keeps order and drops duplicates", func(t *testing.T) {
		assert.Equal(t, []string{"db2", "aws"}, RequiredProviderNames([]*Requirement{
			{Provider: "db2"},
			{Provider: "aws"},
			{Provider: "db2", Id: "a-different-id"},
		}))
	})

	t.Run("skips requirements that are not providers", func(t *testing.T) {
		assert.Equal(t, []string{"aws"}, RequiredProviderNames([]*Requirement{
			nil,
			{Provider: ""},
			{Provider: "aws"},
		}))
	})
}

func TestBundleRequiredProviders(t *testing.T) {
	b := &Bundle{
		Policies: []*Policy{
			{Name: "one", Require: []*Requirement{{Provider: "db2"}}},
			{Name: "two"},
			{Name: "three", Require: []*Requirement{{Provider: "aws"}, {Provider: "db2"}}},
		},
		Packs: []*QueryPack{
			{Name: "pack", Require: []*Requirement{{Provider: "k8s"}, {Provider: "aws"}}},
		},
	}

	// deduplicated across policies and querypacks alike
	assert.Equal(t, []string{"db2", "aws", "k8s"}, b.RequiredProviders())
}

func TestBundleHasRequirements(t *testing.T) {
	assert.False(t, (&Bundle{Policies: []*Policy{{Name: "one"}}}).HasRequirements())
	assert.True(t, (&Bundle{Policies: []*Policy{{Require: []*Requirement{{Provider: "db2"}}}}}).HasRequirements())
	assert.True(t, (&Bundle{Packs: []*QueryPack{{Require: []*Requirement{{Provider: "db2"}}}}}).HasRequirements())
}
