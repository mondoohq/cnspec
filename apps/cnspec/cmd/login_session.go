// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"strings"
	"time"

	"go.mondoo.com/mql/cli/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// sessionTimeLayout is how a login's end is shown, as `cnspec login` shows it.
const sessionTimeLayout = "2006-01-02 15:04 MST"

// credentialsNote names the credentials a command talks to Mondoo Platform
// with. A login made by `cnspec login` is a user, not a service account.
func credentialsNote(opts *config.Config) string {
	if opts == nil || !opts.IsOAuthSession() {
		return "using service account credentials"
	}
	who := "your login"
	if email := opts.Authentication.UserEmail; email != "" {
		who = "the login of " + email
	}
	if until, ok := opts.SessionExpiry(); ok {
		return fmt.Sprintf("using %s, valid until %s", who, until.Local().Format(sessionTimeLayout))
	}
	return "using " + who
}

// loginSession is what a command remembers about the login it runs with, to
// explain a refusal by the server.
type loginSession struct {
	// validUntil is when the login ends. Zero when the config does not say.
	validUntil time.Time
}

// loginSessionOf returns the login the config holds, or nil when its
// credentials are something else.
func loginSessionOf(opts *config.Config) *loginSession {
	if opts == nil || !opts.IsOAuthSession() {
		return nil
	}
	s := &loginSession{}
	s.validUntil, _ = opts.SessionExpiry()
	return s
}

// refusalAdvice says what to do when the server refuses the login: a login
// ends after a limited time and can be ended early on the server. It returns
// "" when there is no login, or when err is not such a refusal.
func (s *loginSession) refusalAdvice(err error, now time.Time) string {
	if s == nil || !isUnauthenticated(err) {
		return ""
	}
	if !s.validUntil.IsZero() && !now.Before(s.validUntil) {
		return fmt.Sprintf("your login ended at %s. Run `cnspec login` to log in again",
			s.validUntil.Local().Format(sessionTimeLayout))
	}
	return "the server no longer accepts your login: it has ended or was revoked. Run `cnspec login` to log in again"
}

// explain returns err with the advice of refusalAdvice in front of it, and
// err unchanged when there is none.
func (s *loginSession) explain(err error, now time.Time) error {
	if advice := s.refusalAdvice(err, now); advice != "" {
		return fmt.Errorf("%s: %w", advice, err)
	}
	return err
}

// isUnauthenticated reports whether err is the server refusing the
// credential. Some callers flatten the error to its text, and the GraphQL
// client reports the HTTP status, so the text is checked too.
func isUnauthenticated(err error) bool {
	if err == nil {
		return false
	}
	if status.Code(err) == codes.Unauthenticated {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "code = Unauthenticated") || strings.Contains(msg, "401 Unauthorized")
}
