// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package credentialcheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func TestValidate_UnsupportedProvider(t *testing.T) {
	res, err := Validate(context.Background(), &inventory.Config{Type: "gcp"})
	require.NoError(t, err)
	require.Equal(t, StateUnknown, res.State)
	require.Contains(t, res.Message, "not supported")
	require.Nil(t, res.ExpiresAt)
}

func TestState_String(t *testing.T) {
	require.Equal(t, "OK", StateOK.String())
	require.Equal(t, "AUTH_ERROR", StateAuthError.String())
	require.Equal(t, "UNKNOWN", StateUnknown.String())
}
