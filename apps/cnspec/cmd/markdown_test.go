// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlainText(t *testing.T) {
	for _, tc := range []struct {
		name string
		md   string
		want string
	}{
		{
			name: "headings lose their markers",
			md:   "### Examples: cloud\n\n#### Scan AWS",
			want: "Examples: cloud\n\nScan AWS",
		},
		{
			// A relative link resolves against the docs site, which a terminal is
			// not sitting in. Left alone it would send the reader to a path that
			// exists nowhere.
			name: "a relative link becomes an address",
			md:   "To learn more, read [Assess AWS Security](/cnspec/cloud/aws).",
			want: "To learn more, read Assess AWS Security (https://mondoo.com/docs/cnspec/cloud/aws).",
		},
		{
			name: "an absolute link is left as it is",
			md:   "See [the schema](https://schema.ocsf.io/) for details.",
			want: "See the schema (https://schema.ocsf.io/) for details.",
		},
		{
			name: "several links on one line",
			md:   "[a](/one) and [b](/two)",
			want: "a (https://mondoo.com/docs/one) and b (https://mondoo.com/docs/two)",
		},
		{
			name: "fenced examples are indented instead",
			md:   "Scan it:\n\n```bash\ncnspec scan local\n```\n",
			want: "Scan it:\n\n  cnspec scan local",
		},
		{
			// The brackets of a placeholder are not a link, and a command that
			// came back mangled would be one someone pasted and ran.
			name: "a command is not rewritten",
			md:   "```bash\ncnspec scan ssh user@host # [see docs]\n```",
			want: "  cnspec scan ssh user@host # [see docs]",
		},
		{
			name: "prose without markup is unchanged",
			md:   "This command triggers a new policy-based scan on an asset.",
			want: "This command triggers a new policy-based scan on an asset.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, plainText(tc.md))
		})
	}
}

// TestPlainTextLeavesNoMarkup is the property that matters for the help text as
// a whole: whatever the docs page gains, the terminal does not show its markers.
func TestPlainTextLeavesNoMarkup(t *testing.T) {
	require.NotEmpty(t, scanCmdDocs, "the scan description has to be embedded")
	help := plainText(scanCmdDocs)

	assert.NotContains(t, help, "```", "no fences")
	assert.NotContains(t, help, "](", "no link syntax")
	for _, line := range strings.Split(help, "\n") {
		assert.False(t, strings.HasPrefix(line, "#"), "no heading markers: %q", line)
	}
	assert.NotContains(t, help, "](/", "no relative links")
	assert.Contains(t, help, "https://mondoo.com/docs/cnspec/",
		"the links have to survive as addresses rather than being dropped")

	// Every target the page documents is a reason someone runs --help.
	for _, target := range []string{
		"cnspec scan local", "cnspec scan aws", "cnspec scan azure", "cnspec scan gcp project",
		"cnspec scan k8s", "cnspec scan oci", "cnspec scan github repo", "cnspec scan gitlab",
		"cnspec scan google-workspace", "cnspec scan atlassian jira", "cnspec scan ms365",
		"cnspec scan okta", "cnspec scan slack", "cnspec scan docker", "cnspec scan vagrant",
		"cnspec scan container registry", "cnspec scan --inventory-file",
	} {
		assert.Contains(t, help, target)
	}
}

// TestScanLongIsTheRenderedDocs pins the direction the two forms are derived in.
// The command carries the rendering; the Markdown is the source, and is what
// GenerateMarkdown puts back for the page.
func TestScanLongIsTheRenderedDocs(t *testing.T) {
	assert.Equal(t, plainText(scanCmdDocs), scanCmd.Long)
}
