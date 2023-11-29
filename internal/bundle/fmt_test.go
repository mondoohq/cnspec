// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleFormatter(t *testing.T) {
	data := `
# This is a comment
policies:
  - uid: sshd-server-policy
    authors:
      - name: Jane Doe
        email: jane@example.com
    tags:
      key: value
      another-key: another-value
    name: SSH Server Policy
    groups:
      - filters: asset.family.contains('unix')
        checks:
          - uid: query1
    version: "1.0.0"
    scoring_system: 2
queries:
  - uid: query1
    docs:
      desc: |
        Secure Boot is required in order to ensure that the booting kernel hasn't been modified. It needs to be enabled in your computer's firmware and be supported by your Linux distribution.
      audit: |
        Run the "mokutil --sb-state" command and check whether it prints "SecureBoot enabled"
      remediation: |
        Enable Secure Boot in your computer's firmware and use a Linux distribution supporting Secure Boot
    mql: |
      command('mokutil --sb-state').stdout.downcase.contains('secureboot enabled')
    impact: 100
    title: Ensure Secure Boot is enabled
`

	b, err := ParseYaml([]byte(data))
	require.NoError(t, err)
	formatted, err := FormatBundle(b, false)
	require.NoError(t, err)

	expected := `# This is a comment
policies:
  - uid: sshd-server-policy
    name: SSH Server Policy
    version: 1.0.0
    tags:
      another-key: another-value
      key: value
    authors:
      - name: Jane Doe
        email: jane@example.com
    groups:
      - filters: asset.family.contains('unix')
        checks:
          - uid: query1
    scoring_system: 2
queries:
  - uid: query1
    title: Ensure Secure Boot is enabled
    impact: 100
    mql: |
      command('mokutil --sb-state').stdout.downcase.contains('secureboot enabled')
    docs:
      desc: |
        Secure Boot is required in order to ensure that the booting kernel hasn't been modified. It needs to be enabled in your computer's firmware and be supported by your Linux distribution.
      audit: |
        Run the "mokutil --sb-state" command and check whether it prints "SecureBoot enabled"
      remediation: |
        Enable Secure Boot in your computer's firmware and use a Linux distribution supporting Secure Boot
`
	assert.Equal(t, expected, string(formatted))
}

func TestBundleSortAndFormat(t *testing.T) {
	data := `
policies:
  - uid: sshd-server-policy
    authors:
      - name: Jane Doe
        email: jane@example.com
    tags:
      key: value
      another-key: another-value
    name: SSH Server Policy
    groups:
      - filters: asset.family.contains('unix')
        checks:
          - uid: query1
            variants:
              - uid: variant2
              - uid: variant1
    version: "1.0.0"
    scoring_system: 2
queries:
  - uid: query2
    variants:
      - uid: variant1
      - uid: variant2
    docs:
      desc: |
        Secure Boot is required in order to ensure that the booting kernel hasn't been modified. It needs to be enabled in your computer's firmware and be supported by your Linux distribution.
      audit: |
        Run the "mokutil --sb-state" command and check whether it prints "SecureBoot enabled"
      remediation: |
        Enable Secure Boot in your computer's firmware and use a Linux distribution supporting Secure Boot
    mql: |
      command('mokutil --sb-state').stdout.downcase.contains('secureboot enabled')
    impact: 100
    title: Ensure Secure Boot is enabled 2
  - uid: query1
    docs:
      desc: |
        Secure Boot is required in order to ensure that the booting kernel hasn't been modified. It needs to be enabled in your computer's firmware and be supported by your Linux distribution.
      audit: |
        Run the "mokutil --sb-state" command and check whether it prints "SecureBoot enabled"
      remediation: |
        Enable Secure Boot in your computer's firmware and use a Linux distribution supporting Secure Boot
    mql: |
      command('mokutil --sb-state').stdout.downcase.contains('secureboot enabled')
    impact: 100
    title: Ensure Secure Boot is enabled
`

	b, err := ParseYaml([]byte(data))
	require.NoError(t, err)
	formatted, err := FormatBundle(b, true)
	require.NoError(t, err)
	expected := `policies:
  - uid: sshd-server-policy
    name: SSH Server Policy
    version: 1.0.0
    tags:
      another-key: another-value
      key: value
    authors:
      - name: Jane Doe
        email: jane@example.com
    groups:
      - filters: asset.family.contains('unix')
        checks:
          - uid: query1
            variants:
              - uid: variant1
              - uid: variant2
    scoring_system: 2
queries:
  - uid: query1
    title: Ensure Secure Boot is enabled
    impact: 100
    mql: |
      command('mokutil --sb-state').stdout.downcase.contains('secureboot enabled')
    docs:
      desc: |
        Secure Boot is required in order to ensure that the booting kernel hasn't been modified. It needs to be enabled in your computer's firmware and be supported by your Linux distribution.
      audit: |
        Run the "mokutil --sb-state" command and check whether it prints "SecureBoot enabled"
      remediation: |
        Enable Secure Boot in your computer's firmware and use a Linux distribution supporting Secure Boot
  - uid: query2
    title: Ensure Secure Boot is enabled 2
    impact: 100
    mql: |
      command('mokutil --sb-state').stdout.downcase.contains('secureboot enabled')
    docs:
      desc: |
        Secure Boot is required in order to ensure that the booting kernel hasn't been modified. It needs to be enabled in your computer's firmware and be supported by your Linux distribution.
      audit: |
        Run the "mokutil --sb-state" command and check whether it prints "SecureBoot enabled"
      remediation: |
        Enable Secure Boot in your computer's firmware and use a Linux distribution supporting Secure Boot
    variants:
      - uid: variant1
      - uid: variant2
`
	assert.Equal(t, expected, string(formatted))
}
