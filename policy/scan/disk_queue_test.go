package scan

import (
	"os"
	"testing"
	"time"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
)

var testQueueConfig = diskQueueConfig{
	dir:         "testdir", // TODO: consider configurable path
	filename:    "disk-queue",
	segmentSize: 500,
	sync:        false,
}

func TestDiskQueueClient_EnqueueDequeue(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-queue")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Update the configuration to use the temporary directory
	testConfig := defaultDqueConfig
	testConfig.dir = tempDir

	handlerCalled := false
	handler := func(job *Job) {
		handlerCalled = true
		// Perform additional checks on job if necessary
	}

	client, err := newDqueClient(testConfig, handler)
	if err != nil {
		t.Fatalf("Failed to create diskQueueClient: %v", err)
	}
	defer client.Stop()

	// Initially, the directory should be empty
	initialFileCount, err := countFilesInDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to count files in temp dir: %v", err)
	}
	if initialFileCount != 0 {
		t.Errorf("Expected 0 files in temp dir, found %d", initialFileCount)
	}

	// Test Enqueue
	testJob := &Job{
		Inventory: &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{
					{
						Connections: []*inventory.Config{
							{
								Type: "k8s",
								Options: map[string]string{
									"path": "./testdata/2pods.yaml",
								},
								Discover: &inventory.Discovery{
									Targets: []string{"auto"},
								},
							},
						},
						ManagedBy: "mondoo-operator-123",
					},
				},
			},
		},
	}
	for i := 1; i < 50; i++ {
		client.Channel() <- *testJob
	}
	// Allow some time to process
	time.Sleep(2 * time.Second)

	if !handlerCalled {
		t.Errorf("Expected handler to be called after dequeue")
	}
}

func countFilesInDir(directory string) (int, error) {
	files, err := os.ReadDir(directory)
	if err != nil {
		return 0, err
	}
	return len(files), nil
}
