// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pdque

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextAvailableFilename(t *testing.T) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "diskqueue_test")
	require.NoError(t, err)
	defer os.RemoveAll(testDir) // Clean up

	// Initialize a new Queue
	q := &Queue{Name: "testQueue", path: testDir}

	var timestamps sync.Map
	var wg sync.WaitGroup
	var mu sync.Mutex // Mutex to protect map operations

	// Use a channel to collect errors from goroutines
	errChan := make(chan error, 1010) // Buffer should be the number of goroutines

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			filename, err := q.nextAvailableFilename()
			if err != nil {
				errChan <- fmt.Errorf("failed to generate filename: %s", err)
				return
			}
			mu.Lock()
			if _, exists := timestamps.Load(filename); exists {
				errChan <- fmt.Errorf("duplicate filename generated: %s", filename)
			} else {
				timestamps.Store(filename, struct{}{})
			}
			mu.Unlock()

			// Create a file to simulate an existing job
			filePath := filepath.Join(testDir, filename+jobFileExt)
			if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
				errChan <- fmt.Errorf("failed to write test file: %s", err)
			}
		}()
	}

	// Test that filenames do not collide in a tight loop
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			filename, err := q.nextAvailableFilename()
			require.NoError(t, err)
			timestamps.Store(filename, struct{}{})
		}()
	}

	// Test if the function properly handles existing files with incremented names
	filename, err := q.nextAvailableFilename()
	require.NoError(t, err)

	// Manually create files that would conflict to ensure our function increments properly
	baseFilename := strings.TrimSuffix(filename, jobFileExt)
	for i := 0; i < 3; i++ {
		conflictFilename := baseFilename + "_" + strconv.Itoa(i) + jobFileExt
		filePath := filepath.Join(testDir, conflictFilename)
		err := os.WriteFile(filePath, []byte("test"), 0o644)
		require.NoError(t, err)
	}

	// Next filename should be incremented
	nextFilename, err := q.nextAvailableFilename()
	require.NoError(t, err)

	if nextFilename == filename {
		t.Errorf("Expected next filename to be different, got: %s", nextFilename)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// Check for any errors sent by goroutines
	for err := range errChan {
		t.Error(err)
	}
}

func TestEnqueue(t *testing.T) {
	// Setup: create a temporary directory to act as the queue directory.
	testDir, err := os.MkdirTemp("", "test_queue")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Instantiate the Queue.
	queue, err := New("testQueue", testDir, 1000, func() interface{} {
		return nil
	})
	require.NoError(t, err)

	testJob := struct {
		Data string
	}{
		Data: "test data",
	}

	// Enqueue the job.
	err = queue.Enqueue([]byte(fmt.Sprintf("%v", testJob)))
	require.NoError(t, err)

	// Verify that a file has been created in the queue directory.
	files, err := os.ReadDir(testDir)
	require.NoError(t, err)

	if len(files) != 1 {
		t.Fatalf("Expected 1 file in queue directory, found %d", len(files))
	}
}

func TestClose(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test_queue")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Create some temporary files to simulate the state of the queue with pending jobs.
	tempFiles := []string{".tmp1", ".tmp2", ".tmp3"}
	for _, f := range tempFiles {
		tmpFilePath := filepath.Join(testDir, f)
		err := os.WriteFile(tmpFilePath, []byte("data"), 0o644)
		require.NoError(t, err)
	}

	// Instantiate the Queue.
	queue, err := New("testQueue", testDir, 1000, func() interface{} {
		return nil
	})
	require.NoError(t, err)

	// Close the queue.
	err = queue.Close()
	require.NoError(t, err)

	// Verify that the queue is marked as closed.
	if !queue.closed {
		t.Errorf("Queue should be marked as closed.")
	}

	// Verify that temporary files are cleaned up.
	files, err := os.ReadDir(testDir)
	require.NoError(t, err)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			t.Errorf("Temporary file %s was not cleaned up", file.Name())
		}
	}

	// Verify that no new actions can be performed on the queue.
	err = queue.Enqueue([]byte("data"))
	require.Error(t, err)

	_, err = queue.Dequeue()
	require.Error(t, err)
}

type testObj struct {
	Name string
	ID   int
}

func TestEnqueueDequeue(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test_enqueue_dequeue")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)
	// Setup: Initialize the queue
	queue, err := NewOrOpen("testQueue", testDir, 10, func() interface{} {
		return new(testObj)
	})
	require.NoError(t, err)
	defer queue.Close()

	// Test enqueue
	testObj := &testObj{Name: "test"}
	err = queue.Enqueue(testObj)
	require.NoError(t, err)

	// Test dequeue
	dequeuedObj, err := queue.Dequeue()
	require.NoError(t, err)

	assert.Equal(t, testObj, dequeuedObj)
}

func TestQueueMaxSize(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test_maxSize")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)
	maxSize := 5
	queue, err := NewOrOpen("testQueue", testDir, maxSize, func() interface{} {
		return new(testObj)
	})
	require.NoError(t, err)
	defer queue.Close()

	// Enqueue items up to the maximum size
	for i := 0; i < maxSize; i++ {
		err := queue.Enqueue(&testObj{Name: fmt.Sprintf("test%d", i)})
		require.NoError(t, err)
	}

	// Attempt to enqueue one more item, which should fail
	err = queue.Enqueue(&testObj{Name: "overflow"})
	require.Error(t, err)

	if !errors.Is(err, ErrQueueFull) {
		t.Errorf("Expected ErrQueueFull, but got %v", err)
	}

	// Dequeue an item
	_, err = queue.Dequeue()
	require.NoError(t, err)

	// Attempt to enqueue again, which should now succeed
	err = queue.Enqueue(&testObj{Name: "shouldSucceed"})
	require.NoError(t, err)
}

// TestEnqueueDequeue tests enqueuing and dequeuing of jobs
func TestEnqueueDequeueMore(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test_enqueue_dequeue")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Create a new queue
	q, err := NewOrOpen("testQueue", testDir, 1000, func() interface{} { return new(testObj) })
	require.NoError(t, err)
	defer q.Close()

	// Enqueue 1000 jobs
	for i := 0; i < 1000; i++ {
		err := q.Enqueue(&testObj{ID: i})
		require.NoError(t, err)
	}

	// Verify there are 1000 job files
	jobCount, err := countJobFiles(testDir)
	require.NoError(t, err)

	require.Equal(t, jobCount, 1000)

	// Dequeue and check each job
	for i := 0; i < 1000; i++ {
		obj, err := q.Dequeue()
		require.NoError(t, err)

		job, ok := obj.(*testObj)
		if !ok {
			t.Fatalf("Dequeued object is not of type *TestJob")
		}

		assert.Equal(t, job.ID, i)
	}

	if obj, _ := q.Dequeue(); obj != nil {
		t.Errorf("Expected queue to be empty, but got a job")
	}
}

// countJobFiles counts the number of job files in the given directory
func countJobFiles(dir string) (int, error) {
	var count int
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(info.Name()) == jobFileExt {
			count++
		}
		return nil
	})
	return count, err
}

// TestConcurrentEnqueueDequeue tests concurrent enqueuing and dequeuing of jobs
func TestConcurrentEnqueueDequeue(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test_enqueue_dequeue")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	const numJobs = 200

	// Create a new queue
	q, err := NewOrOpen("testConcurrentQueue", testDir, numJobs, func() interface{} { return &testObj{} })
	require.NoError(t, err)
	defer q.Close()

	var wg sync.WaitGroup

	// Concurrently enqueue jobs
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < (numJobs / 2); i++ {
			err := q.Enqueue(&testObj{ID: i})
			require.NoError(t, err)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 100; i < numJobs; i++ {
			err := q.Enqueue(&testObj{ID: i})
			require.NoError(t, err)
		}
	}()

	// Concurrently dequeue jobs
	dequeuedJobs := make(map[int]bool)
	var mu sync.Mutex
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numJobs; i++ {
			obj, err := q.DequeueBlock()
			require.NoError(t, err)

			job, ok := obj.(*testObj)
			if !ok {
				t.Errorf("Dequeued object is not of type *TestJob")
				continue
			}

			mu.Lock()
			dequeuedJobs[job.ID] = true
			mu.Unlock()
		}
	}()

	wg.Wait()

	// Verify all jobs were dequeued
	assert.Equal(t, len(dequeuedJobs), numJobs)
}
