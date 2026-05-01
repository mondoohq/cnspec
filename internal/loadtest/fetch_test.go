// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loadtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGCSURI(t *testing.T) {
	cases := []struct {
		uri        string
		wantBucket string
		wantPrefix string
		wantErr    bool
	}{
		{"gs://my-bucket/seeds/", "my-bucket", "seeds/", false},
		{"gs://my-bucket/seeds", "my-bucket", "seeds", false},
		{"gs://my-bucket/", "my-bucket", "", false},
		{"gs://my-bucket", "my-bucket", "", false},
		{"gs://", "", "", true},
		{"gs:///prefix", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.uri, func(t *testing.T) {
			b, p, err := parseGCSURI(tc.uri)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantBucket, b)
			require.Equal(t, tc.wantPrefix, p)
		})
	}
}

func TestMaterializeSeedsLocalPathPassthrough(t *testing.T) {
	// Local paths must short-circuit before any GCS client setup so the
	// existing local-only workflow never accidentally requires ADC.
	dir := t.TempDir()
	got, cleanup, err := MaterializeSeeds(context.Background(), dir)
	require.NoError(t, err)
	defer cleanup()
	require.Equal(t, dir, got)
}
