// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func testAsset(name, platform string) *inventory.Asset {
	a := &inventory.Asset{Name: name}
	if platform != "" {
		a.Platform = &inventory.Platform{Name: platform}
	}
	return a
}

func TestTodoListSingleAsset(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	tl, err := newTodoListProgram(&in, &buf, WithScore())
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Millisecond)
		tl.AddTask("1", testAsset("test1", "linux"))
		tl.OnProgress("1", 0.5)
		tl.OnProgress("1", 1.0)
		tl.Score("1", "A")
		tl.Completed("1")
		tl.Close()
	}()
	err = tl.Open()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "test1")
	assert.Contains(t, output, "[linux]")
	assert.Contains(t, output, "A")
	assert.Contains(t, output, "1/1 completed")
}

func TestTodoListMultipleAssets(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	tl, err := newTodoListProgram(&in, &buf, WithScore())
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Millisecond)
		tl.AddTask("1", testAsset("test1", "linux"))
		tl.AddTask("2", testAsset("test2", "aws"))
		tl.AddTask("3", testAsset("test3", "k8s"))
		tl.OnProgress("1", 1.0)
		tl.Score("1", "A")
		tl.Completed("1")
		tl.OnProgress("2", 1.0)
		tl.Score("2", "B")
		tl.Completed("2")
		tl.OnProgress("3", 1.0)
		tl.Score("3", "F")
		tl.Completed("3")
		tl.Close()
	}()
	err = tl.Open()
	require.NoError(t, err)

	output := buf.String()
	// Rolling window: final view shows last 2 finished tasks
	assert.Contains(t, output, "test2")
	assert.Contains(t, output, "test3")
	assert.Contains(t, output, "F")
	assert.Contains(t, output, "3/3 completed")
}

func TestTodoListErrored(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	tl, err := newTodoListProgram(&in, &buf, WithScore())
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Millisecond)
		tl.AddTask("1", testAsset("test1", "linux"))
		tl.AddTask("2", testAsset("test2", "aws"))
		tl.AddTask("3", testAsset("test3", "k8s"))
		tl.OnProgress("1", 1.0)
		tl.Score("1", "A")
		tl.Completed("1")
		tl.Score("2", "X")
		tl.Errored("2")
		tl.OnProgress("3", 1.0)
		tl.Score("3", "F")
		tl.Completed("3")
		tl.Close()
	}()
	err = tl.Open()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "✗")
	assert.Contains(t, output, "3/3 completed")
	assert.Contains(t, output, "1 errored")
}

func TestTodoListNotApplicable(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	tl, err := newTodoListProgram(&in, &buf, WithScore())
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Millisecond)
		tl.AddTask("1", testAsset("test1", "linux"))
		tl.AddTask("2", testAsset("test2", "aws"))
		tl.AddTask("3", testAsset("test3", "k8s"))
		tl.OnProgress("1", 1.0)
		tl.Score("1", "A")
		tl.Completed("1")
		tl.Score("2", "X")
		tl.Errored("2")
		tl.Score("3", "U")
		tl.NotApplicable("3")
		tl.Close()
	}()
	err = tl.Open()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "3/3 completed")
	assert.Contains(t, output, "1 errored")
	assert.NotContains(t, output, "n/a")
}

func TestTodoListOnlyOneErrored(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	tl, err := newTodoListProgram(&in, &buf)
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Millisecond)
		tl.AddTask("1", testAsset("test1", "linux"))
		tl.Errored("1")
		tl.Close()
	}()
	err = tl.Open()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "✗")
	assert.Contains(t, output, "test1")
	assert.Contains(t, output, "1/1 completed")
	assert.Contains(t, output, "1 errored")
}

func TestTodoListDynamicAddition(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	tl, err := newTodoListProgram(&in, &buf, WithScore())
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Millisecond)
		// Add first task, start scanning
		tl.AddTask("1", testAsset("test1", "linux"))
		tl.OnProgress("1", 1.0)
		tl.Score("1", "A")
		// Add second task before completing first (simulates batch scan)
		tl.AddTask("2", testAsset("test2", "aws"))
		tl.Completed("1")
		tl.OnProgress("2", 1.0)
		tl.Score("2", "B")
		tl.Completed("2")
		tl.Close()
	}()
	err = tl.Open()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "test1")
	assert.Contains(t, output, "test2")
	assert.Contains(t, output, "2/2 completed")
}

func TestTodoListCloseBeforeAllDone(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	tl, err := newTodoListProgram(&in, &buf)
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Millisecond)
		tl.AddTask("1", testAsset("test1", "linux"))
		tl.AddTask("2", testAsset("test2", "aws"))
		tl.OnProgress("1", 0.5)
		tl.Completed("1")
		// Close without completing task 2
		tl.Close()
	}()
	err = tl.Open()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Scanning assets...")
}

func TestTodoListMoreAssetsOverflow(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	tl, err := newTodoListProgram(&in, &buf)
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Millisecond)
		for i := 1; i <= 10; i++ {
			tl.AddTask(fmt.Sprintf("%d", i), testAsset(fmt.Sprintf("test%d", i), "linux"))
		}
		// Complete first one, leave rest pending
		tl.Completed("1")
		tl.Close()
	}()
	err = tl.Open()
	require.NoError(t, err)

	output := buf.String()
	// 1 finished + 4 pending shown, 5 remaining
	assert.Contains(t, output, "+5 more...")
}

// TestTodoListMultiBatch verifies that completing a first batch of tasks does
// not close the progress bar before a second batch is added. This is the
// regression test for the >50 assets bug where allDoneLocked() would quit the
// UI between batches.
func TestTodoListMultiBatch(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	tl, err := newTodoListProgram(&in, &buf, WithScore())
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Millisecond)

		// Batch 1: add and complete 3 tasks
		tl.AddTask("1", testAsset("batch1-a", "linux"))
		tl.AddTask("2", testAsset("batch1-b", "linux"))
		tl.AddTask("3", testAsset("batch1-c", "linux"))
		tl.Completed("1")
		tl.Completed("2")
		tl.Completed("3")

		// Simulate the gap between batches — previously allDoneLocked()
		// would have returned true here and quit the program.
		time.Sleep(5 * time.Millisecond)

		// Batch 2: add and complete 2 more tasks
		tl.AddTask("4", testAsset("batch2-a", "aws"))
		tl.AddTask("5", testAsset("batch2-b", "aws"))
		tl.Completed("4")
		tl.Completed("5")
		tl.Close()
	}()
	err = tl.Open()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "batch2-b")
	assert.Contains(t, output, "5/5 completed")
}

func TestTodoListDiscoveredAndSkipped(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	tl, err := newTodoListProgram(&in, &buf)
	require.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Millisecond)
		tl.Discovered(100)
		tl.AddTask("1", testAsset("scannable1", "linux"))
		tl.AddTask("2", testAsset("scannable2", "linux"))
		tl.Filtered(3)
		tl.Completed("1")
		tl.Completed("2")
		tl.Close()
	}()
	err = tl.Open()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "100 discovered")
	assert.Contains(t, output, "2 scanned")
	assert.Contains(t, output, "3 filtered")
}
