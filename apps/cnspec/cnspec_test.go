// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSelfUpdateExemptCommand(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		// nothing to inspect
		{[]string{"cnspec"}, false},
		{nil, false},

		// help and version, wherever they appear
		{[]string{"cnspec", "--help"}, true},
		{[]string{"cnspec", "-h"}, true},
		{[]string{"cnspec", "help"}, true},
		{[]string{"cnspec", "help", "scan"}, true},
		{[]string{"cnspec", "scan", "--help"}, true},
		{[]string{"cnspec", "scan", "docker", "-h"}, true},
		{[]string{"cnspec", "policy", "list", "--help"}, true},
		{[]string{"cnspec", "version"}, true},
		{[]string{"cnspec", "--version"}, true},

		// the explicit update command runs the same check itself
		{[]string{"cnspec", "update"}, true},
		{[]string{"cnspec", "update", "--channel", "preview"}, true},

		// enrollment commands, including their aliases
		{[]string{"cnspec", "login"}, true},
		{[]string{"cnspec", "login", "--token", "abc"}, true},
		{[]string{"cnspec", "register"}, true},
		{[]string{"cnspec", "logout"}, true},
		{[]string{"cnspec", "logout", "--force"}, true},
		{[]string{"cnspec", "unregister"}, true},

		// everything else still updates
		{[]string{"cnspec", "scan", "local"}, false},
		{[]string{"cnspec", "shell", "local"}, false},
		{[]string{"cnspec", "status"}, false},
		// a positional that merely reads like an exempt command is not one
		{[]string{"cnspec", "scan", "ssh", "login@example.com"}, false},
	}

	for _, test := range tests {
		t.Run(joinArgs(test.args), func(t *testing.T) {
			assert.Equal(t, test.want, isSelfUpdateExemptCommand(test.args))
		})
	}
}

func joinArgs(args []string) string {
	if len(args) == 0 {
		return "<empty>"
	}
	out := args[0]
	for _, a := range args[1:] {
		out += " " + a
	}
	return out
}
