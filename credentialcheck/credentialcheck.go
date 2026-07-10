// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package credentialcheck validates an integration's stored credentials by making
// one cheap authenticated provider call (no scan, no plugin, no resource enumeration).
package credentialcheck

import (
	"context"
	"fmt"
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// State describes the outcome of a credential validation attempt.
type State int

const (
	// StateUnknown is returned when the provider has no validator, or when the
	// validation attempt could not conclusively determine whether the
	// credential is valid (e.g. network errors, throttling).
	StateUnknown State = iota
	// StateOK means the credential successfully authenticated.
	StateOK
	// StateAuthError means the provider rejected the credential.
	StateAuthError
)

// String implements fmt.Stringer for State.
func (s State) String() string {
	switch s {
	case StateOK:
		return "OK"
	case StateAuthError:
		return "AUTH_ERROR"
	default:
		return "UNKNOWN"
	}
}

// Result is the outcome of a credential validation.
type Result struct {
	State     State
	Message   string
	ExpiresAt *time.Time // set only when a provider exposes credential expiry; nil for AWS
}

// Validate connects with the credentials in conf and performs the minimum
// authenticated call that proves they are valid. It never runs a scan or spawns
// a provider plugin. Providers without a validator return StateUnknown.
func Validate(ctx context.Context, conf *inventory.Config) (Result, error) {
	switch conf.GetType() {
	case "aws":
		return validateAWS(ctx, conf), nil
	default:
		return Result{
			State:   StateUnknown,
			Message: fmt.Sprintf("credential validation not supported for provider %q", conf.GetType()),
		}, nil
	}
}
