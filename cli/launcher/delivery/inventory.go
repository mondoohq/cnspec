// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package delivery

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// When a form holds a credential, the launcher stops assembling a command line
// and writes an inventory file instead.
//
// The reason is narrow and non-negotiable: the launcher runs the command by
// re-executing cnspec, and anything on that command line is visible to every
// user on the machine through `ps auxww`. A password typed into a form must not
// end up there. An inventory file, written 0600 in a private directory and
// deleted after the run, keeps it out of the process table.

// Everything the launcher writes to disk is registered here, so that one call
// removes all of it.
//
// The inventory carries a plaintext credential whenever the OS keychain was
// unavailable, and it used to be removed on exactly one event: the command it
// fed reporting that it had finished. That covers the happy path and nothing
// else. Quitting the launcher before running the scan, or a signal, or a panic,
// each left a file holding a password in the system temp directory, for as long
// as the machine went without a reboot.
//
// Unlink-after-open, which is the usual answer, is not available here: the path
// is handed to a child process, which opens it by name, so the file has to keep
// one for as long as the child might start. What is available is to make every
// exit this process can observe remove it -- the command finishing, quitting,
// Run returning at all, a panic unwinding through it, and SIGHUP -- and that is
// what this and CleanupTempFiles do. SIGKILL and losing power are observable by
// nobody, and remain the residual risk; the file is 0600 inside a 0700
// directory, which is what limits the damage there.
var pendingTemp struct {
	mu   sync.Mutex
	next int
	fns  map[int]func()
}

// TrackTemp registers a cleanup and returns it wrapped so that running it --
// however it is reached -- also unregisters it. The wrapper is safe to call
// more than once, which matters because both the launch path and the exit path
// legitimately hold one.
func TrackTemp(cleanup func()) func() {
	if cleanup == nil {
		return func() {}
	}
	pendingTemp.mu.Lock()
	if pendingTemp.fns == nil {
		pendingTemp.fns = map[int]func(){}
	}
	id := pendingTemp.next
	pendingTemp.next++
	pendingTemp.fns[id] = cleanup
	pendingTemp.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			pendingTemp.mu.Lock()
			delete(pendingTemp.fns, id)
			pendingTemp.mu.Unlock()
			cleanup()
		})
	}
}

// CleanupTempFiles removes everything the launcher has written and not yet
// cleaned up. Calling it when there is nothing outstanding is free.
func CleanupTempFiles() {
	pendingTemp.mu.Lock()
	fns := make([]func(), 0, len(pendingTemp.fns))
	for id, fn := range pendingTemp.fns {
		fns = append(fns, fn)
		delete(pendingTemp.fns, id)
	}
	pendingTemp.mu.Unlock()

	for _, fn := range fns {
		fn()
	}
}

// RenderInventory marshals the inventory and checks that cnspec can read what
// came out.
//
// Marshalling happens before PreProcess, which rewrites credentials into
// spec.credentials under generated ids; validation runs on a copy so the bytes
// stay in the readable shape while still being checked. Validating through the
// same path `cnspec scan` uses is what makes a bad reference surface here
// rather than as a connect-time failure later.
func RenderInventory(inv *inventory.Inventory) ([]byte, error) {
	data, err := inv.ToYAML()
	if err != nil {
		return nil, errors.Wrap(err, "cannot render the inventory")
	}

	check, err := inventory.InventoryFromYAML(data)
	if err != nil {
		return nil, errors.Wrap(err, "generated an unreadable inventory")
	}
	if err := check.PreProcess(); err != nil {
		return nil, errors.Wrap(err, "generated an invalid inventory")
	}
	if err := check.Validate(); err != nil {
		return nil, errors.Wrap(err, "generated an invalid inventory")
	}
	return data, nil
}

// ExportInventory writes the inventory to a path the user chose.
//
// Three things separate this from WriteInventory, and all three follow from the
// file being the user's rather than the launcher's:
//
//   - It is not registered with TrackTemp. Every other file this package writes
//     is removed on the way out; this one is the artifact the user asked for,
//     and an export that deleted itself when the launcher exited would be no
//     export at all.
//   - O_EXCL, so an export never replaces a file that is already there. The
//     launcher cannot tell an accidental collision from a deliberate overwrite,
//     and the destructive reading of that ambiguity is the one that cannot be
//     undone.
//   - It is still 0600. The permissions do not relax just because the path did:
//     an exported inventory either references a keychain entry or, for the
//     connectors whose provider keeps the secret in conn.Options, carries the
//     secret itself, and neither is anybody else's business.
func ExportInventory(inv *inventory.Inventory, path string) error {
	data, err := RenderInventory(inv)
	if err != nil {
		return err
	}

	fh, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return errors.Newf("%s already exists, remove it or choose another name", path)
		}
		return errors.Wrapf(err, "cannot create %s", path)
	}
	if _, err := fh.Write(data); err != nil {
		_ = fh.Close()
		// Removed rather than left behind: a half-written inventory under the
		// name the user chose is a file a scheduled scan would later read as
		// though it were complete.
		_ = os.Remove(path)
		return errors.Wrapf(err, "cannot write %s", path)
	}
	if err := fh.Close(); err != nil {
		_ = os.Remove(path)
		return errors.Wrapf(err, "cannot write %s", path)
	}
	return nil
}

// WriteInventory marshals the inventory to a private 0600 file and returns its
// path along with a cleanup func.
func WriteInventory(inv *inventory.Inventory) (path string, cleanup func(), err error) {
	data, err := RenderInventory(inv)
	if err != nil {
		return "", nil, err
	}

	// A private directory, not just a private file: on some systems the file
	// name alone leaks, and 0700 on the directory keeps other users out.
	dir, err := os.MkdirTemp("", "cnspec-ui-")
	if err != nil {
		return "", nil, errors.Wrap(err, "cannot create a directory for the inventory")
	}
	// Registered before the file is written, not after: from here on there is
	// something on disk, and every way out of the process removes it.
	cleanup = TrackTemp(func() { _ = os.RemoveAll(dir) })

	path = filepath.Join(dir, "inventory.yml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		cleanup()
		return "", nil, errors.Wrap(err, "cannot write the inventory")
	}
	return path, cleanup, nil
}
