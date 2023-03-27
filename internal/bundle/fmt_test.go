package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleFormatter(t *testing.T) {
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
    specs:
      - asset_filter:
          query: platform.family.contains(_ == 'unix')
        scoring_queries:
          query1:
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
    query: |
      command('mokutil --sb-state').stdout.downcase.contains('secureboot enabled')
    severity: 100
    title: Ensure Secure Boot is enabled
`

	formatted, err := DeprecatedV7_ToV8([]byte(data))
	require.NoError(t, err)

	expected := `policies:
  - uid: sshd-server-policy
    name: SSH Server Policy
    version: 1.0.0
    license: unspecified
    tags:
      another-key: another-value
      key: value
    authors:
      - name: Jane Doe
        email: jane@example.com
    groups:
      - filters: platform.family.contains(_ == 'unix')
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
