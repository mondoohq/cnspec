// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"os"
	"testing"
	"time"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
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