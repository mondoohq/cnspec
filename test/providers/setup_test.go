// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"log"
	"os/exec"
)

// setup builds cnspec locally
func setup() {
	// build cnspec
	if err := exec.Command("go", "build", "../../apps/cnspec/cnspec.go").Run(); err != nil {
		log.Fatalf("building cnspec: %v", err)
	}
}
