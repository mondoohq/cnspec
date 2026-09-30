// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fence = "```"

func TestParseCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     []CodeBlock
	}{
		{
			name:     "no code",
			markdown: "Disable the service.",
			want:     nil,
		},
		{
			name:     "language and untagged blocks",
			markdown: "Run:\n\n" + fence + "bash\nset -e\necho hi\n" + fence + "\n\nThen:\n\n" + fence + "\n# modprobe -r cramfs\n" + fence + "\n",
			want: []CodeBlock{
				{Lang: "bash", Code: "set -e\necho hi\n"},
				{Lang: "", Code: "# modprobe -r cramfs\n"},
			},
		},
		{
			name:     "info string keeps only the language",
			markdown: fence + " sh title=fix.sh\nls\n" + fence,
			want:     []CodeBlock{{Lang: "sh", Code: "ls\n"}},
		},
		{
			name:     "indented fence inside a list item",
			markdown: "1. Edit the file:\n\n   " + fence + "ini\n   [main]\n     nested = 1\n   " + fence + "\n",
			want:     []CodeBlock{{Lang: "ini", Code: "[main]\n  nested = 1\n"}},
		},
		{
			name:     "longer fence holds a shorter one",
			markdown: "````markdown\n" + fence + "bash\nls\n" + fence + "\n````\n",
			want:     []CodeBlock{{Lang: "markdown", Code: fence + "bash\nls\n" + fence + "\n"}},
		},
		{
			name:     "tilde fence",
			markdown: "~~~powershell\nGet-Service\n~~~\n",
			want:     []CodeBlock{{Lang: "powershell", Code: "Get-Service\n"}},
		},
		{
			name:     "unclosed block runs to the end",
			markdown: fence + "bash\necho open",
			want:     []CodeBlock{{Lang: "bash", Code: "echo open\n"}},
		},
		{
			name:     "inline code is not a fence",
			markdown: "Run " + fence + "ls" + fence + " now.",
			want:     nil,
		},
		{
			name:     "empty block",
			markdown: fence + "bash\n" + fence,
			want:     []CodeBlock{{Lang: "bash", Code: ""}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseCodeBlocks(tt.markdown))
		})
	}
}

func TestPolicyGraph_Remediations(t *testing.T) {
	data := []byte(`
policies:
  - uid: linux-security
    name: Linux Security
    groups:
      - title: Kernel
        checks:
          - uid: cramfs-disabled
          - uid: core-dumps
          - uid: no-remediation
queries:
  - uid: cramfs-disabled
    title: Ensure cramfs is disabled
    mql: kernel.module("cramfs").loaded == false
    docs:
      remediation:
        - id: cli
          desc: |
            Unload the module:

            ` + fence + `
            modprobe -r cramfs
            ` + fence + `
        - id: bash
          desc: |
            ` + fence + `bash
            #!/bin/bash
            modprobe -r cramfs
            ` + fence + `
  - uid: core-dumps
    title: Ensure core dumps are restricted
    mql: "true"
    docs:
      remediation:
        - id: bash
          desc: |
            ` + fence + `sh
            sysctl -w fs.suid_dumpable=0
            ` + fence + `
        - id: ansible
          desc: |
            ` + fence + `yaml
            - hosts: all
            ` + fence + `
  - uid: no-remediation
    title: No remediation
    mql: "true"
`)
	b, err := ParseYaml(data)
	require.NoError(t, err)
	g := BuildGraph(map[string]*Bundle{"test.mql.yaml": b})

	node := func(name string) string {
		for _, n := range g.Nodes {
			if n.Name == name {
				return n.ID
			}
		}
		t.Fatalf("node %s not found", name)
		return ""
	}

	t.Run("single check returns all entries with parsed code", func(t *testing.T) {
		res := g.Remediations(node("cramfs-disabled"), RemediationOpts{})
		require.Len(t, res, 1)
		assert.Equal(t, "cramfs-disabled", res[0].UID)
		assert.Equal(t, "Ensure cramfs is disabled", res[0].Title)
		require.Len(t, res[0].Remediations, 2)
		assert.Equal(t, "cli", res[0].Remediations[0].ID)
		assert.Equal(t, []CodeBlock{{Code: "modprobe -r cramfs\n"}}, res[0].Remediations[0].Code)
		assert.Equal(t, []CodeBlock{{Lang: "bash", Code: "#!/bin/bash\nmodprobe -r cramfs\n"}}, res[0].Remediations[1].Code)
	})

	t.Run("id filter", func(t *testing.T) {
		res := g.Remediations(node("cramfs-disabled"), RemediationOpts{IDs: []string{"BASH"}})
		require.Len(t, res, 1)
		require.Len(t, res[0].Remediations, 1)
		assert.Equal(t, "bash", res[0].Remediations[0].ID)
	})

	t.Run("policy expands to its checks and skips those without remediations", func(t *testing.T) {
		res := g.Remediations(node("linux-security"), RemediationOpts{IDs: []string{"bash"}})
		var uids []string
		for _, r := range res {
			uids = append(uids, r.UID)
		}
		assert.ElementsMatch(t, []string{"cramfs-disabled", "core-dumps"}, uids)
	})

	t.Run("lang filter drops entries without a matching block", func(t *testing.T) {
		res := g.Remediations(node("linux-security"), RemediationOpts{Langs: []string{"sh"}})
		require.Len(t, res, 1)
		assert.Equal(t, "core-dumps", res[0].UID)
		require.Len(t, res[0].Remediations, 1)
		assert.Equal(t, []CodeBlock{{Lang: "sh", Code: "sysctl -w fs.suid_dumpable=0\n"}}, res[0].Remediations[0].Code)
	})

	t.Run("check without remediations", func(t *testing.T) {
		assert.Empty(t, g.Remediations(node("no-remediation"), RemediationOpts{}))
	})
}
