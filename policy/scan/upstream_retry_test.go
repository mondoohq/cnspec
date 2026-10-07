// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/discovery"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// fakeClock is a clock the retry's sleeps advance, so the tests take no time.
type fakeClock struct{ t time.Time }

func testUpstreamRetry(maxWait time.Duration) (*upstreamRetry, *fakeClock) {
	c := &fakeClock{t: time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)}
	r := newUpstreamRetry()
	r.maxWait = maxWait
	r.now = func() time.Time { return c.t }
	r.sleep = func(ctx context.Context, d time.Duration) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		c.t = c.t.Add(d)
		return nil
	}
	return r, c
}

// failing returns fn that fails with err for its first n calls, then succeeds,
// and the count of calls made.
func failing(n int, err error) (func() error, *int) {
	calls := 0
	return func() error {
		calls++
		if calls <= n {
			return err
		}
		return nil
	}, &calls
}

// What a scan sees while the server restarts behind a load balancer, and when
// nothing listens at all (a raw transport error, which carries no status).
var (
	errUnavailable = status.Error(codes.Unavailable, "503 Service Unavailable")
	errRefused     = errors.New("failed to do request: dial tcp 10.0.0.1:443: connect: connection refused")
)

func TestUpstreamRetryWaitsOutARestart(t *testing.T) {
	for name, transient := range map[string]error{"unavailable": errUnavailable, "refused": errRefused} {
		t.Run(name, func(t *testing.T) {
			r, c := testUpstreamRetry(defaultUpstreamWait)
			start := c.t
			// Six failed attempts wait at most 1+2+4+8+15+15 = 45s.
			fn, calls := failing(6, transient)
			require.NoError(t, r.do(context.Background(), "GetBundle", fn))
			assert.Equal(t, 7, *calls)
			assert.Less(t, c.t.Sub(start), defaultUpstreamWait)
		})
	}
}

func TestUpstreamRetryPermanentErrorFailsAtOnce(t *testing.T) {
	r, c := testUpstreamRetry(defaultUpstreamWait)
	start := c.t
	denied := status.Error(codes.PermissionDenied, "request permission denied")
	fn, calls := failing(10, denied)
	err := r.do(context.Background(), "GetBundle", fn)
	require.ErrorIs(t, err, denied)
	assert.Equal(t, 1, *calls)
	assert.Equal(t, start, c.t, "no wait")
}

// A platform that stays down: the first call waits up to maxWait and gives up;
// later calls of the same scan try once, until a call succeeds again.
func TestUpstreamRetryGivesUpThenFailsFastUntilTheNextSuccess(t *testing.T) {
	r, c := testUpstreamRetry(defaultUpstreamWait)
	start := c.t
	fn, calls := failing(1000, errUnavailable)
	require.ErrorIs(t, r.do(context.Background(), "SynchronizeAssets", fn), errUnavailable)
	assert.Greater(t, *calls, 5)
	assert.LessOrEqual(t, c.t.Sub(start), defaultUpstreamWait)

	fn, calls = failing(1000, errUnavailable)
	require.ErrorIs(t, r.do(context.Background(), "ResolveAndUpdateJobs", fn), errUnavailable)
	assert.Equal(t, 1, *calls, "the platform is known to be down: one try")

	fn, _ = failing(0, nil)
	require.NoError(t, r.do(context.Background(), "ResolveAndUpdateJobs", fn))

	fn, calls = failing(2, errUnavailable)
	require.NoError(t, r.do(context.Background(), "ResolveAndUpdateJobs", fn))
	assert.Equal(t, 3, *calls, "a success restored the full wait")
}

func TestUpstreamRetryStopsWhenTheScanIsCancelled(t *testing.T) {
	r, _ := testUpstreamRetry(defaultUpstreamWait)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fn, calls := failing(1000, errUnavailable)
	require.ErrorIs(t, r.do(ctx, "GetBundle", fn), errUnavailable)
	assert.Equal(t, 1, *calls)
}

func TestUpstreamRetryIsSharedThroughTheContext(t *testing.T) {
	r := newUpstreamRetry()
	ctx := withUpstreamRetry(context.Background(), r)
	assert.Same(t, r, upstreamRetryFrom(ctx))
	assert.Nil(t, upstreamRetryFrom(context.Background()), "no retry in the context: the retry is off")
}

// Off unless MONDOO_UPSTREAM_RETRY says otherwise, with MONDOO_WINDOWS_NATIVE's
// values.
func TestUpstreamRetryEnv(t *testing.T) {
	for v, want := range map[string]bool{
		"": false, "0": false, "false": false, "FALSE": false, "no": false, "off": false, " off ": false,
		"1": true, "true": true, "on": true, "yes": true,
	} {
		t.Setenv(UpstreamRetryEnv, v)
		assert.Equal(t, want, upstreamRetryEnabled(), "%q", v)
	}
}

// Without a retry (MONDOO_UPSTREAM_RETRY unset), a call runs once, even on a
// transient error.
func TestUpstreamRetryOffRunsOnce(t *testing.T) {
	var r *upstreamRetry
	fn, calls := failing(1000, errUnavailable)
	require.ErrorIs(t, r.do(context.Background(), "GetBundle", fn), errUnavailable)
	assert.Equal(t, 1, *calls)
}

func TestUpstreamRetryWait(t *testing.T) {
	for attempt, want := range map[int]time.Duration{1: time.Second, 2: 2 * time.Second, 3: 4 * time.Second, 4: 8 * time.Second, 5: 15 * time.Second, 60: 15 * time.Second} {
		for range 20 {
			got := upstreamRetryWait(attempt)
			assert.GreaterOrEqual(t, got, want/2, "attempt %d", attempt)
			assert.LessOrEqual(t, got, want, "attempt %d", attempt)
		}
	}
}

// flakySync fails SynchronizeAssets with err for its first fails calls.
type flakySync struct {
	*activityRecorder
	fails, calls int
	err          error
}

func (f *flakySync) SynchronizeAssets(ctx context.Context, req *policy.SynchronizeAssetsReq) (*policy.SynchronizeAssetsResp, error) {
	f.calls++
	if f.calls <= f.fails {
		return nil, f.err
	}
	return f.activityRecorder.SynchronizeAssets(ctx, req)
}

// With the retry on, asset synchronization waits out a restart through the
// scan's retry, and a permanent error fails it at once. With it off, it keeps
// its own three attempts on any error.
func TestSyncBatchRetriesThroughTheScanRetry(t *testing.T) {
	batch := []*discovery.TrackedAsset{{Asset: &inventory.Asset{Name: "a"}}}
	trigger := policy.AssetActivityTrigger_ASSET_ACTIVITY_TRIGGER_AD_HOC

	r, _ := testUpstreamRetry(defaultUpstreamWait)
	ctx := withUpstreamRetry(context.Background(), r)
	f := &flakySync{activityRecorder: newActivityRecorder(), fails: 3, err: errUnavailable}
	require.NoError(t, syncBatchWithUpstream(ctx, batch, &policy.Services{PolicyResolver: f}, "//spaces/s", nil, trigger))
	assert.Equal(t, 4, f.calls)
	f.next(t)

	r, _ = testUpstreamRetry(defaultUpstreamWait)
	ctx = withUpstreamRetry(context.Background(), r)
	denied := status.Error(codes.PermissionDenied, "request permission denied")
	f = &flakySync{activityRecorder: newActivityRecorder(), fails: 3, err: denied}
	require.Error(t, syncBatchWithUpstream(ctx, batch, &policy.Services{PolicyResolver: f}, "//spaces/s", nil, trigger))
	assert.Equal(t, 1, f.calls)

	if testing.Short() {
		return // the legacy loop waits 2s and 4s
	}
	f = &flakySync{activityRecorder: newActivityRecorder(), fails: 5, err: denied}
	require.Error(t, syncBatchWithUpstream(context.Background(), batch, &policy.Services{PolicyResolver: f}, "//spaces/s", nil, trigger))
	assert.Equal(t, 3, f.calls, "retry off: three attempts, as before")
}
