// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package delivery

import (
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
)

// The wait for the keychain is bounded, and the bound is not the context.
//
// A locked keychain raises an OS dialog and the write does not return until it
// is answered; the keyring backends reach the OS through blocking calls no
// context can interrupt, so a timeout that lived only in the context would be
// a timeout that never fires. This one holds regardless of what the backend
// does, which is what lets the launcher fall back to the 0600 inventory rather
// than wait for a dialog nobody is looking at.
func TestTheKeychainWaitIsBounded(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	// A write that never answers, which is what a locked keychain waiting on a
	// dialog behind the launcher looks like from here.
	never := func(string, *vault.Credential) error {
		<-release
		return nil
	}

	start := time.Now()
	err := StoreCredentialWithin(20*time.Millisecond, never, "cnspec-ui-test",
		&vault.Credential{Type: vault.CredentialType_password, Password: "<PLACEHOLDER-not-a-real-secret>"})
	if err == nil {
		t.Fatal("a keychain that never answered was reported as a success")
	}
	if waited := time.Since(start); waited > 5*time.Second {
		t.Fatalf("the wait was not bounded: %s", waited)
	}
	// The message has to point at the dialog, because the dialog is behind the
	// full-screen UI and is the thing the user has to go and find.
	if !strings.Contains(err.Error(), "keychain") || !strings.Contains(err.Error(), "locked") {
		t.Errorf("the timeout should say where to look, got %q", err)
	}
}

// A keychain that answers is not slowed down by the bound, and its own error
// is passed through rather than replaced by a timeout.
func TestTheKeychainAnswerIsPassedThrough(t *testing.T) {
	answers := func(string, *vault.Credential) error { return nil }
	if err := StoreCredentialWithin(KeychainTimeout, answers, "id", nil); err != nil {
		t.Fatalf("a successful write reported %v", err)
	}

	refuses := func(string, *vault.Credential) error {
		return errors.New("keyring unavailable")
	}
	err := StoreCredentialWithin(KeychainTimeout, refuses, "id", nil)
	if err == nil || !strings.Contains(err.Error(), "keyring unavailable") {
		t.Fatalf("the backend's own reason was lost: %v", err)
	}
}
