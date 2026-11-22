// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

type fakeVulnFetcher struct {
	calls   atomic.Int32
	answers func(call int) (*policy.GetUploadVulnResultResp, error)
}

func (f *fakeVulnFetcher) GetUploadVulnResult(ctx context.Context, req *policy.GetUploadVulnResultReq) (*policy.GetUploadVulnResultResp, error) {
	return f.answers(int(f.calls.Add(1)))
}

func fastPolling(t *testing.T) {
	first, max := vulnResultsFirstPoll, vulnResultsMaxPoll
	vulnResultsFirstPoll, vulnResultsMaxPoll = time.Millisecond, 2*time.Millisecond
	t.Cleanup(func() { vulnResultsFirstPoll, vulnResultsMaxPoll = first, max })
}

var vulnReq = &policy.GetUploadVulnResultReq{UploadSessionId: "session", ScopeMrn: "//asset"}

func TestAwaitVulnResultPollsUntilReady(t *testing.T) {
	fastPolling(t)
	report := &mvd.VulnReport{Stats: &mvd.ReportStats{Score: 98}}
	f := &fakeVulnFetcher{answers: func(call int) (*policy.GetUploadVulnResultResp, error) {
		if call < 3 {
			return &policy.GetUploadVulnResultResp{State: policy.VulnResultState_VULN_RESULT_STATE_PENDING}, nil
		}
		return &policy.GetUploadVulnResultResp{State: policy.VulnResultState_VULN_RESULT_STATE_READY, Report: report, NewerScan: true}, nil
	}}

	assessment, got := awaitVulnResult(context.Background(), f, vulnReq, time.Minute)
	assert.Equal(t, policy.VulnResultState_VULN_RESULT_STATE_READY, assessment.State)
	assert.True(t, assessment.NewerScan)
	assert.Same(t, report, got)
	assert.Equal(t, int32(3), f.calls.Load())
}

func TestAwaitVulnResultTimesOutWithoutData(t *testing.T) {
	fastPolling(t)
	f := &fakeVulnFetcher{answers: func(int) (*policy.GetUploadVulnResultResp, error) {
		return &policy.GetUploadVulnResultResp{State: policy.VulnResultState_VULN_RESULT_STATE_PENDING}, nil
	}}

	assessment, got := awaitVulnResult(context.Background(), f, vulnReq, 20*time.Millisecond)
	assert.Nil(t, got)
	assert.Equal(t, policy.VulnResultState_VULN_RESULT_STATE_PENDING, assessment.State)
	assert.Contains(t, assessment.Message, "within 20ms")
}

func TestAwaitVulnResultUpstreamWithoutRPC(t *testing.T) {
	for _, code := range []codes.Code{codes.NotFound, codes.Unimplemented} {
		f := &fakeVulnFetcher{answers: func(int) (*policy.GetUploadVulnResultResp, error) {
			return nil, status.Error(code, "unknown method")
		}}
		assessment, got := awaitVulnResult(context.Background(), f, vulnReq, time.Minute)
		assert.Nil(t, got)
		assert.Equal(t, policy.VulnResultState_VULN_RESULT_STATE_NOT_ASSESSED, assessment.State, code.String())
		assert.Equal(t, int32(1), f.calls.Load(), "no retries for %s", code)
	}
}

func TestAwaitVulnResultRetriesErrorsAndReportsTheLast(t *testing.T) {
	fastPolling(t)
	f := &fakeVulnFetcher{answers: func(int) (*policy.GetUploadVulnResultResp, error) {
		return nil, errors.New("connection reset")
	}}

	assessment, got := awaitVulnResult(context.Background(), f, vulnReq, 20*time.Millisecond)
	assert.Nil(t, got)
	assert.Equal(t, policy.VulnResultState_VULN_RESULT_STATE_FAILED, assessment.State)
	assert.Equal(t, "connection reset", assessment.Message)
	assert.Greater(t, f.calls.Load(), int32(1))
}

func TestAwaitVulnResultDecidedWithoutReport(t *testing.T) {
	for _, state := range []policy.VulnResultState{
		policy.VulnResultState_VULN_RESULT_STATE_NOT_ASSESSED,
		policy.VulnResultState_VULN_RESULT_STATE_FAILED,
	} {
		f := &fakeVulnFetcher{answers: func(int) (*policy.GetUploadVulnResultResp, error) {
			return &policy.GetUploadVulnResultResp{State: state, Message: "why"}, nil
		}}
		assessment, got := awaitVulnResult(context.Background(), f, vulnReq, time.Minute)
		require.Nil(t, got)
		assert.Equal(t, state, assessment.State)
		assert.Equal(t, "why", assessment.Message)
	}
}
