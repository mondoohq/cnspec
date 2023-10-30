// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package serve

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
)

var once sync.Once

// setup builds cnspec locally
func setup() {
	if err := exec.Command("go", "build", "../../apps/cnspec/cnspec.go").Run(); err != nil {
		log.Fatalf("building cnspec: %v", err)
	}
}

func TestService(t *testing.T) {
	// We need to build the daemon first, since go run does not forward the SIGTERM signal
	once.Do(setup)

	// Start the "daemon" process
	cmd := exec.Command("./cnspec", []string{
		"serve",
	}...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting the daemon: %s\n", err)
		return
	}

	fmt.Printf("Daemon started with PID %d\n", cmd.Process.Pid)

	// Give it a moment to initialize
	time.Sleep(5 * time.Second)

	// Send a SIGTERM signal to the daemon for a graceful shutdown
	fmt.Println("Sending SIGTERM to the daemon...")
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("Error sending SIGTERM: %s\n", err)
		return
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Command finished with error: %v\n", err)
	}

	// Check the output
	fmt.Println("Daemon stdout:\n", stdout.String())
	fmt.Println("Daemon stderr:\n", stderr.String())

	// Validate the output
	expectedText := "bye bye space cowboy"
	if bytes.Contains(stderr.Bytes(), []byte(expectedText)) {
		fmt.Println("Success: Output contains the expected text.")
	} else {
		fmt.Println("Failure: Output does not contain the expected text.")
	}

}
