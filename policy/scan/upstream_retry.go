// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/providers-sdk/v1/upstream"
)

// Waiting out an unreachable Mondoo Platform. The calls a scan cannot do
// without (the space bundle, asset synchronization, policy resolution) used to
// fail the scan on the first transient error, or after a few seconds of
// retries. A server restart or a rolling deployment takes longer than that, so
// a scan that started at the wrong moment failed and waited for the next one.
const (
	// defaultUpstreamWait is how long one call keeps retrying a transient
	// error: long enough to outlast a server restart.
	defaultUpstreamWait = 2 * time.Minute

	upstreamRetryBaseWait = time.Second
	upstreamRetryMaxWait  = 15 * time.Second
)

// upstreamRetry retries the Mondoo Platform calls a scan depends on, on
// transient errors only (upstream.RetryableRPCError: unavailable, deadline
// exceeded, resource exhausted, aborted, and raw network failures). A
// permanent error (unauthenticated, permission denied, invalid argument, not
// found) returns at once.
//
// Each call may wait up to maxWait. Once one call has waited that long and
// still failed, the platform is treated as down: later calls try once, so a
// scan of many assets fails in minutes rather than spending maxWait on every
// asset. The first call that succeeds lifts that again.
//
// Only idempotent calls belong here: a call that failed with a transient error
// may have reached the server.
type upstreamRetry struct {
	maxWait time.Duration
	// sleep waits for d or until ctx ends; tests replace it.
	sleep func(ctx context.Context, d time.Duration) error
	now   func() time.Time

	mu   sync.Mutex
	down bool
}

func newUpstreamRetry() *upstreamRetry {
	return &upstreamRetry{maxWait: defaultUpstreamWait, sleep: sleepCtx, now: time.Now}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

type upstreamRetryKey struct{}

// withUpstreamRetry attaches r to ctx, so every call of one scan shares it.
func withUpstreamRetry(ctx context.Context, r *upstreamRetry) context.Context {
	return context.WithValue(ctx, upstreamRetryKey{}, r)
}

// upstreamRetryFrom returns the scan's upstreamRetry, or a new one for a call
// made outside a scan.
func upstreamRetryFrom(ctx context.Context) *upstreamRetry {
	if r, ok := ctx.Value(upstreamRetryKey{}).(*upstreamRetry); ok && r != nil {
		return r
	}
	return newUpstreamRetry()
}

// do runs fn, and runs it again on a transient error until it succeeds, fails
// permanently, or the call's wait is used up. what names the call in the log.
func (r *upstreamRetry) do(ctx context.Context, what string, fn func() error) error {
	r.mu.Lock()
	down := r.down
	r.mu.Unlock()

	deadline := r.now().Add(r.maxWait)
	if down {
		deadline = r.now()
	}
	for attempt := 1; ; attempt++ {
		err := fn()
		if err == nil {
			r.setDown(false)
			return nil
		}
		if !upstream.RetryableRPCError(err) {
			return err
		}
		wait := upstreamRetryWait(attempt)
		if r.now().Add(wait).After(deadline) {
			r.setDown(true)
			return err
		}
		log.Warn().Err(err).Str("call", what).Int("attempt", attempt).Dur("retry_in", wait).
			Msg("could not reach Mondoo Platform, retrying")
		if serr := r.sleep(ctx, wait); serr != nil {
			return err
		}
	}
}

func (r *upstreamRetry) setDown(down bool) {
	r.mu.Lock()
	r.down = down
	r.mu.Unlock()
}

// upstreamRetryWait is the wait before attempt+1: doubling from
// upstreamRetryBaseWait up to upstreamRetryMaxWait, with jitter in its upper
// half so a fleet of scans does not retry in step.
func upstreamRetryWait(attempt int) time.Duration {
	d := upstreamRetryMaxWait
	if attempt < 8 {
		d = min(upstreamRetryBaseWait<<(attempt-1), upstreamRetryMaxWait)
	}
	return d/2 + time.Duration(rand.Int64N(int64(d/2)+1)) //nolint:gosec // jitter, not security-sensitive
}
