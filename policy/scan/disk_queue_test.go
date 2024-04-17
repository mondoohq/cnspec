// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"os"
	"testing"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
)

func TestDiskQueueClient_EnqueueDequeue(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Update the configuration to use the temporary directory
	testConfig := defaultDqueConfig
	testConfig.dir = tempDir

	completionChannel := make(chan struct{}, 50) // Channel to signal job completion

	handler := func(job *Job) {
		completionChannel <- struct{}{} // Signal completion
	}

	client, err := newDqueClient(testConfig, handler)
	if err != nil {
		t.Fatalf("Failed to create diskQueueClient: %v", err)
	}
	defer client.Stop()

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
	for i := 0; i < 50; i++ {
		client.Channel() <- *testJob
	}

	for i := 0; i < 50; i++ {
		<-completionChannel
	}

	// Verify that all jobs have been processed
	if len(completionChannel) != 0 {
		t.Errorf("Expected handler to be called 50 times, but was called %d times", 50-len(completionChannel))
	}
}
