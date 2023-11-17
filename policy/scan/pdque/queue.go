package pdque

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/afero"
)

const (
	jobFileExt     = ".job"
	tempFileExt    = ".tmp"
	tempFilePrefix = "."
)

// ErrQueueClosed is the error returned when a queue is closed.
var ErrQueueClosed = errors.New("queue is closed")

var ErrQueueFull = errors.New("queue is full")

// ErrUnableToDecode is returned when an object cannot be decoded.
type ErrUnableToDecode struct {
	Path string
	Err  error
}

func (e ErrUnableToDecode) Error() string {
	return fmt.Sprintf("object in file %s cannot be decoded: %s", e.Path, e.Err)
}

type Queue struct {
	Name    string
	path    string
	mu      sync.Mutex
	closed  bool
	cond    *sync.Cond
	builder func() interface{}
	maxSize int
}

func New(name string, path string, maxSize int, builder func() interface{}) (*Queue, error) {
	overlyPermissive, err := isOverlyPermissive(path)
	if err != nil {
		return nil, err
	}
	if overlyPermissive {
		return nil, errors.New("path is overly permissive, make sure it is not writable to others or the group: " + path)
	}

	que := &Queue{
		Name:    name,
		path:    path,
		builder: builder,
		maxSize: maxSize,
	}
	que.cond = sync.NewCond(&que.mu)

	return que, nil
}

func NewOrOpen(name string, path string, maxSize int, builder func() interface{}) (*Queue, error) {
	var que *Queue
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return nil, err
		}
		que, err = New(name, path, maxSize, builder)
		if err != nil {
			return nil, err
		}
	} else {
		que, err = Open(name, path, maxSize, builder)
		if err != nil {
			return nil, err
		}
	}

	return que, nil
}

func Open(name string, path string, maxSize int, builder func() interface{}) (*Queue, error) {
	overlyPermissive, err := isOverlyPermissive(path)
	if err != nil {
		return nil, err
	}
	if overlyPermissive {
		return nil, errors.New("path is overly permissive, make sure it is not writable to others or the group: " + path)
	}

	que := &Queue{
		Name:    name,
		path:    path,
		builder: builder,
		maxSize: maxSize,
	}
	que.cond = sync.NewCond(&que.mu)

	return que, nil
}

// Close safely shuts down the queue, ensuring all resources are released.
func (q *Queue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// If the queue is already closed, return an error or just exit.
	if q.closed {
		return ErrQueueClosed
	}

	// Clean up temporary files.
	files, err := os.ReadDir(q.path)
	if err != nil {
		return err
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), tempFilePrefix) {
			err := os.Remove(filepath.Join(q.path, file.Name()))
			if err != nil {
				return err
			}
		}
	}

	// Set the queue as closed to prevent further operations.
	q.closed = true

	// Wake up all goroutines waiting on the condition variable before closing the queue.
	q.cond.Broadcast()

	return nil
}

func (q *Queue) Enqueue(obj interface{}) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	// Check if the queue size has reached the maxSize
	if q.maxSize > 0 {
		size, err := q.currentSize()
		if err != nil {
			return err // Handle or return the error
		}
		if size >= q.maxSize {
			return ErrQueueFull // Or handle as needed
		}
	}

	// Find the next available filename
	filename, err := q.nextAvailableFilename()
	if err != nil {
		return err
	}

	tempPath := filepath.Join(q.path, tempFilePrefix+filename+tempFileExt)
	finalPath := filepath.Join(q.path, filename+jobFileExt)

	// Encode the struct to a byte buffer
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(obj); err != nil {
		return err
	}

	// Write to a temporary file
	err = os.WriteFile(tempPath, buff.Bytes(), 0644)
	if err != nil {
		return err
	}

	// Rename the temporary file to its final name
	err = os.Rename(tempPath, finalPath)
	if err != nil {
		return err
	}

	// After successfully enqueueing a job, wake up one of the waiting goroutines, if any.
	q.cond.Signal()

	return nil
}

func (q *Queue) Dequeue() (interface{}, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil, ErrQueueClosed
	}

	files, err := os.ReadDir(q.path)
	if err != nil {
		return nil, err
	}

	// Sort job files by name (which is the timestamp)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, file := range files {
		if filepath.Ext(file.Name()) == jobFileExt {
			jobPath := filepath.Join(q.path, file.Name())

			data, err := os.ReadFile(jobPath)
			if err != nil {
				return nil, err
			}

			// Decode the bytes into an object
			obj := q.builder()
			if err := gob.NewDecoder(bytes.NewReader(data)).Decode(obj); err != nil {
				return nil, ErrUnableToDecode{
					Path: q.path,
					Err:  err,
				}
			}

			// Remove the job file
			if err := os.Remove(jobPath); err != nil {
				return nil, err
			}

			return obj, nil
		}
	}

	return nil, errors.New("no jobs in queue")
}

func (q *Queue) DequeueBlock() (interface{}, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		if q.closed {
			return nil, ErrQueueClosed
		}

		files, err := os.ReadDir(q.path)
		if err != nil {
			return nil, err
		}

		// Sort job files by name (which is the timestamp)
		sort.Slice(files, func(i, j int) bool {
			return files[i].Name() < files[j].Name()
		})

		for _, file := range files {
			if filepath.Ext(file.Name()) == jobFileExt {
				jobPath := filepath.Join(q.path, file.Name())

				data, err := os.ReadFile(jobPath)
				if err != nil {
					return nil, err
				}

				// Decode the bytes into an object
				obj := q.builder()
				if err := gob.NewDecoder(bytes.NewReader(data)).Decode(obj); err != nil {
					return nil, ErrUnableToDecode{
						Path: q.path,
						Err:  err,
					}
				}

				// Remove the job file
				if err := os.Remove(jobPath); err != nil {
					return nil, err
				}

				return obj, nil
			}
		}

		// No jobs in queue, wait for a new job to be enqueued
		q.cond.Wait()
	}
}

func (q *Queue) nextAvailableFilename() (string, error) {
	timestamp := time.Now().UnixNano()
	filename := strconv.FormatInt(timestamp, 10)
	for {
		_, err := os.Stat(filepath.Join(q.path, filename+jobFileExt))
		if os.IsNotExist(err) {
			break
		} else if err != nil {
			return "", err
		} else {
			timestamp++
			filename = strconv.FormatInt(timestamp, 10)
		}
	}
	return filename, nil
}

// check the currentSize of the queue
// We do a lot of disk operations here, could find a more performant approach
func (q *Queue) currentSize() (int, error) {
	files, err := os.ReadDir(q.path)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) == jobFileExt {
			count++
		}
	}
	return count, nil
}

func isOverlyPermissive(path string) (bool, error) {
	fs := afero.NewOsFs()
	stat, err := fs.Stat(path)
	if err != nil {
		return true, errors.New("failed to analyze " + path)
	}
	mode := stat.Mode()
	if mode&0o022 != 0 {
		return true, nil
	}
	return false, nil
}