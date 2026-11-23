// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// A refused annotation leaves the asset; one the platform stored stays, and so
// does one whose value no longer matches what was refused.
func TestDropRejectedAnnotations(t *testing.T) {
	asset := &inventory.Asset{
		Name: "a",
		Annotations: map[string]string{
			"team":        "red",
			"environment": "prod",
			"owner":       "alice",
		},
	}

	dropRejectedAnnotations(asset, []*policy.RejectedAnnotation{
		{Key: "team", Value: "red", Reason: "the space does not allow the value \"red\""},
		{Key: "owner", Value: "bob"},
		{Key: "missing", Value: "x"},
	})

	assert.Equal(t, map[string]string{"environment": "prod", "owner": "alice"}, asset.Annotations)
}

// A server that refuses nothing, or predates the field, changes nothing.
func TestDropRejectedAnnotationsNone(t *testing.T) {
	asset := &inventory.Asset{Name: "a", Annotations: map[string]string{"team": "red"}}
	dropRejectedAnnotations(asset, nil)
	assert.Equal(t, map[string]string{"team": "red"}, asset.Annotations)

	empty := &inventory.Asset{Name: "b"}
	dropRejectedAnnotations(empty, []*policy.RejectedAnnotation{{Key: "team", Value: "red"}})
	assert.Empty(t, empty.Annotations)
}
