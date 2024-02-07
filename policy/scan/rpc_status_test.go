// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package scan

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/ranger-rpc/codes"
)

func TestRPCStatus(t *testing.T) {
	t.Run("with unknown error", func(t *testing.T) {
		err := errors.New("unknown error")
		s := rpcStatus(err)
		assert.Equal(t, codes.Unknown, s.Code())
		assert.Equal(t, "unknown error", s.Message())
	})

	t.Run("with wrapped RPC error", func(t *testing.T) {
		err := errors.New("rpc error: code = Unimplemented desc = platform vulnerabilities for test are not supported")
		s := rpcStatus(err)
		assert.Equal(t, codes.Unimplemented, s.Code())
		assert.Equal(t, "platform vulnerabilities for test are not supported", s.Message())
	})

	t.Run("with other wrapped RPC error", func(t *testing.T) {
		err := errors.New("rpc error: code = NotFound desc = resource not found")
		s := rpcStatus(err)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "resource not found", s.Message())
	})
}
