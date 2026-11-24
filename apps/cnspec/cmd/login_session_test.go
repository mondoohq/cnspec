// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/cli/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func loginConfig(email string, validUntil string) *config.Config {
	opts := &config.Config{}
	opts.Authentication = &config.CliConfigAuthentication{
		Method:     config.AUTH_METHOD_OAUTH,
		UserEmail:  email,
		ValidUntil: validUntil,
	}
	return opts
}

func TestCredentialsNote(t *testing.T) {
	until := time.Date(2030, 1, 2, 15, 4, 0, 0, time.UTC)

	t.Run("a service account", func(t *testing.T) {
		assert.Equal(t, "using service account credentials", credentialsNote(&config.Config{}))
		assert.Equal(t, "using service account credentials", credentialsNote(nil))
	})

	t.Run("a login names its user and its end", func(t *testing.T) {
		got := credentialsNote(loginConfig("user@example.com", until.Format(time.RFC3339)))
		assert.Equal(t, "using the login of user@example.com, valid until "+until.Local().Format(sessionTimeLayout), got)
		assert.NotContains(t, got, "service account")
	})

	t.Run("a login without a recorded user or end", func(t *testing.T) {
		assert.Equal(t, "using your login", credentialsNote(loginConfig("", "")))
	})
}

func TestRefusalAdvice(t *testing.T) {
	now := time.Date(2030, 1, 2, 12, 0, 0, 0, time.UTC)
	refused := status.Error(codes.Unauthenticated, "request permission unauthenticated")

	t.Run("no login, no advice", func(t *testing.T) {
		assert.Nil(t, loginSessionOf(&config.Config{}))
		assert.Nil(t, loginSessionOf(nil))
		var none *loginSession
		assert.Empty(t, none.refusalAdvice(refused, now))
	})

	t.Run("a login that is still valid was ended on the server", func(t *testing.T) {
		s := loginSessionOf(loginConfig("user@example.com", now.Add(time.Hour).Format(time.RFC3339)))
		require.NotNil(t, s)
		got := s.refusalAdvice(refused, now)
		assert.Contains(t, got, "ended or was revoked")
		assert.Contains(t, got, "`cnspec login`")
	})

	t.Run("a login past its end says when it ended", func(t *testing.T) {
		ended := now.Add(-time.Minute)
		s := loginSessionOf(loginConfig("user@example.com", ended.Format(time.RFC3339)))
		got := s.refusalAdvice(refused, now)
		assert.Contains(t, got, "your login ended at "+ended.Local().Format(sessionTimeLayout))
		assert.Contains(t, got, "`cnspec login`")
	})

	t.Run("a wrapped or flattened refusal is recognized", func(t *testing.T) {
		s := loginSessionOf(loginConfig("user@example.com", ""))
		assert.NotEmpty(t, s.refusalAdvice(fmt.Errorf("failed to sync: %w", refused), now))
		assert.NotEmpty(t, s.refusalAdvice(errors.New(refused.Error()), now))
	})

	t.Run("the GraphQL client's refusal is recognized", func(t *testing.T) {
		s := loginSessionOf(loginConfig("user@example.com", ""))
		gqlErr := errors.New(`non-200 OK status code: 401 Unauthorized body: "{\"code\":16}"`)
		assert.NotEmpty(t, s.refusalAdvice(gqlErr, now))
	})

	t.Run("explain keeps the server's error behind the advice", func(t *testing.T) {
		s := loginSessionOf(loginConfig("user@example.com", ""))
		got := s.explain(refused, now)
		require.ErrorIs(t, got, refused)
		assert.True(t, strings.HasPrefix(got.Error(), "the server no longer accepts your login"), got.Error())

		other := errors.New("connection refused")
		assert.Same(t, other, s.explain(other, now))
		var none *loginSession
		assert.Same(t, refused, none.explain(refused, now))
	})

	t.Run("another error gets no advice", func(t *testing.T) {
		s := loginSessionOf(loginConfig("user@example.com", ""))
		assert.Empty(t, s.refusalAdvice(nil, now))
		assert.Empty(t, s.refusalAdvice(errors.New("connection refused"), now))
		assert.Empty(t, s.refusalAdvice(status.Error(codes.PermissionDenied, "no valid permissions"), now))
	})
}
