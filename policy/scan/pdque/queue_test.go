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
)

func TestNextAvailableFilename(t *testing.T) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "diskqueue_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(testDir) // Clean up

	// Initialize a new Queue
	q := &Queue{Name: "testQueue", path: testDir}

	// Test that filenames are generated with increasing timestamps
	var timestamps sync.Map
	var wg sync.WaitGroup
	var mu sync.Mutex // Mutex to protect map operations

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			filename, err := q.nextAvailableFilename()
			if err != nil {
				t.Errorf("Failed to generate filename: %s", err)
			}
			mu.Lock()
			if _, exists := timestamps.Load(filename); exists {
				t.Errorf("Duplicate filename generated: %s", filename)
			}
			timestamps.Store(filename, struct{}{})
			mu.Unlock()

			// Create a file to simulate an existing job
			filePath := filepath.Join(testDir, filename+jobFileExt)
			if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
				t.Errorf("Failed to write test file: %s", err)
			}
		}()
	}

	// Test that filenames do not collide in a tight loop
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			filename, err := q.nextAvailableFilename()
			if err != nil {
				t.Errorf("Failed to generate filename in goroutine: %s", err)
			}
			timestamps.Store(filename, struct{}{})
		}()
	}

	// Test if the function properly handles existing files with incremented names
	filename, err := q.nextAvailableFilename()
	if err != nil {
		t.Fatalf("Failed to generate filename for increment test: %s", err)
	}

	// Manually create files that would conflict to ensure our function increments properly
	baseFilename := strings.TrimSuffix(filename, jobFileExt)
	for i := 0; i < 3; i++ {
		conflictFilename := baseFilename + "_" + strconv.Itoa(i) + jobFileExt
		filePath := filepath.Join(testDir, conflictFilename)
		if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
			t.Fatalf("Failed to write conflict file: %s", err)
		}
	}

	// Next filename should be incremented
	nextFilename, err := q.nextAvailableFilename()
	if err != nil {
		t.Fatalf("Failed to generate next filename: %s", err)
	}

	if nextFilename == filename {
		t.Errorf("Expected next filename to be different, got: %s", nextFilename)
	}
}

func TestEnqueue(t *testing.T) {
	// Setup: create a temporary directory to act as the queue directory.
	testDir, err := os.MkdirTemp("", "test_queue")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Instantiate the Queue.
	queue, err := New("testQueue", testDir, 1000, func() interface{} {
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}

	// Define a test job (could be any struct or type that you will be enqueueing).
	testJob := struct {
		Data string
	}{
		Data: "test data",
	}

	// Enqueue the job.
	err = queue.Enqueue([]byte(fmt.Sprintf("%v", testJob)))
	if err != nil {
		t.Errorf("Failed to enqueue job: %v", err)
	}

	// Verify that a file has been created in the queue directory.
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read queue directory: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file in queue directory, found %d", len(files))
	}
}

func TestClose(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test_queue")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create some temporary files to simulate the state of the queue with pending jobs.
	tempFiles := []string{".tmp1", ".tmp2", ".tmp3"}
	for _, f := range tempFiles {
		tmpFilePath := filepath.Join(testDir, f)
		if err := os.WriteFile(tmpFilePath, []byte("data"), 0o644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
	}

	// Instantiate the Queue.
	queue, err := New("testQueue", testDir, 1000, func() interface{} {
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}

	// Close the queue.
	if err := queue.Close(); err != nil {
		t.Errorf("Failed to close queue: %v", err)
	}

	// Verify that the queue is marked as closed.
	if !queue.closed {
		t.Errorf("Queue should be marked as closed.")
	}

	// Verify that temporary files are cleaned up.
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read queue directory: %v", err)
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			t.Errorf("Temporary file %s was not cleaned up", file.Name())
		}
	}

	// Verify that no new actions can be performed on the queue.
	if err := queue.Enqueue([]byte("data")); err == nil {
		t.Errorf("Enqueue should fail on a closed queue")
	}

	// Verify that the Dequeue method also behaves as expected.
	if _, err := queue.Dequeue(); err == nil {
		t.Errorf("Dequeue should fail on a closed queue")
	}
}

type testObj struct {
	Name string
	ID   int
}

func TestEnqueueDequeue(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test_enqueue_dequeue")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)
	// Setup: Initialize the queue
	queue, err := NewOrOpen("testQueue", testDir, 10, func() interface{} {
		return new(testObj)
	})
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Close()

	// Test enqueue
	testObj := &testObj{Name: "test"}
	if err := queue.Enqueue(testObj); err != nil {
		t.Errorf("Failed to enqueue object: %v", err)
	}

	// Test dequeue
	dequeuedObj, err := queue.Dequeue()
	if err != nil {
		t.Errorf("Failed to dequeue object: %v", err)
	}

	assert.Equal(t, testObj, dequeuedObj)
}

func TestQueueMaxSize(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test_maxSize")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)
	maxSize := 5
	queue, err := NewOrOpen("testQueue", testDir, maxSize, func() interface{} {
		return new(testObj)
	})
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Close()

	// Enqueue items up to the maximum size
	for i := 0; i < maxSize; i++ {
		if err := queue.Enqueue(&testObj{Name: fmt.Sprintf("test%d", i)}); err != nil {
			t.Fatalf("Failed to enqueue object: %v", err)
		}
	}

	// Attempt to enqueue one more item, which should fail
	err = queue.Enqueue(&testObj{Name: "overflow"})
	if err == nil {
		t.Errorf("Expected an error when enqueuing an item after reaching max size, but got none")
	}

	if !errors.Is(err, ErrQueueFull) {
		t.Errorf("Expected ErrQueueFull, but got %v", err)
	}

	// Dequeue an item
	_, err = queue.Dequeue()
	if err != nil {
		t.Fatalf("Failed to dequeue object: %v", err)
	}

	// Attempt to enqueue again, which should now succeed
	err = queue.Enqueue(&testObj{Name: "shouldSucceed"})
	if err != nil {
		t.Errorf("Failed to enqueue object after dequeuing: %v", err)
	}
}

// TestEnqueueDequeue tests enqueuing and dequeuing of jobs
func TestEnqueueDequeueMore(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test_enqueue_dequeue")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a new queue
	q, err := NewOrOpen("testQueue", testDir, 1000, func() interface{} { return new(testObj) })
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	// Enqueue 1000 jobs
	for i := 0; i < 1000; i++ {
		err := q.Enqueue(&testObj{ID: i})
		if err != nil {
			t.Fatalf("Failed to enqueue job %d: %v", i, err)
		}
	}

	// Verify there are 1000 job files
	jobCount, err := countJobFiles(testDir)
	if err != nil {
		t.Fatalf("Failed to count job files: %v", err)
	}
	if jobCount != 1000 {
		t.Errorf("Expected 1000 job files, found %d", jobCount)
	}

	// Dequeue and check each job
	for i := 0; i < 1000; i++ {
		obj, err := q.Dequeue()
		if err != nil {
			t.Fatalf("Failed to dequeue job %d: %v", i, err)
		}

		job, ok := obj.(*testObj)
		if !ok {
			t.Fatalf("Dequeued object is not of type *TestJob")
		}

		// Additional check: you might want to ensure that the dequeued job has the expected ID
		if job.ID != i {
			t.Errorf("Dequeued job has ID %d; want %d", job.ID, i)
		}
	}

	// Optionally, verify the queue is empty now
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
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	const numJobs = 1000

	// Create a new queue
	q, err := NewOrOpen("testConcurrentQueue", testDir, numJobs, func() interface{} { return &testObj{} })
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	var wg sync.WaitGroup

	// Concurrently enqueue jobs
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numJobs; i++ {
			err := q.Enqueue(&testObj{ID: i})
			if err != nil {
				t.Errorf("Failed to enqueue job %d: %v", i, err)
			}
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
			if err != nil {
				t.Errorf("Failed to dequeue job: %v", err)
				continue
			}

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
	if len(dequeuedJobs) != numJobs {
		t.Errorf("Not all jobs were dequeued: expected %d, got %d", numJobs, len(dequeuedJobs))
	}
}
