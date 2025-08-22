// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tooling

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v12/policy"
)

func TestBundleFormattedYaml(t *testing.T) {
	bundleStr := `
  owner_mrn: //test.sth
  policies:
  - uid: policy1
    groups:
    - filters: "true"
      checks:
      - uid: check-1
        mql: 1 == 2
      - uid: check-2
        mql: failme.name != ""
      queries:
      - uid: query-1
        mql: 1 == 1
      - mql: failme.name
        uid: query-2
`

	bundle, err := policy.BundleFromYAML([]byte(bundleStr))
	require.NoError(t, err)
	require.NotNil(t, bundle)

	data, err := bundle.ToYAML()
	require.NoError(t, err)

	formatted, err := Format(data)
	require.NoError(t, err)

	expectedStr := `owner_mrn: //test.sth
policies:
  - uid: policy1
    groups:
      - filters: "true"
        checks:
          - uid: check-1
            mql: 1 == 2
          - uid: check-2
            mql: failme.name != ""
        queries:
          - uid: query-1
            mql: 1 == 1
          - uid: query-2
            mql: failme.name
`
	require.Equal(t, expectedStr, string(formatted))
}
