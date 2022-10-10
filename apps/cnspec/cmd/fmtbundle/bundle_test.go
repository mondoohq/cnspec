package fmtbundle

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	data := `
queries:
  - uid: mondoo-client-linux-security-baseline-1.2_ensure_secure_boot_is_enabled
    title: Ensure Secure Boot is enabled
    severity: 100
    docs:
      desc: |
        Secure Boot is required in order to ensure that the booting kernel hasn't been modified. It needs to be enabled in your computer's firmware and be supported by your Linux distribution.
      audit: |
        Run the "mokutil --sb-state" command and check whether it prints "SecureBoot enabled"
      remediation: |
        Enable Secure Boot in your computer's firmware and use a Linux distribution supporting Secure Boot
    query: |
      command('mokutil --sb-state').stdout.downcase.contains('secureboot enabled')
`

	baseline, err := ParseYaml([]byte(data))
	require.NoError(t, err)
	assert.NotNil(t, baseline)
	assert.Equal(t, 1, len(baseline.Queries))
	assert.Equal(t, int64(100), baseline.Queries[0].Severity)
}
