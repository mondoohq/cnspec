// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// The poll interval starts at vulnResultsFirstPoll and doubles up to
// vulnResultsMaxPoll. Variables so tests can shorten them.
var (
	vulnResultsFirstPoll = 2 * time.Second
	vulnResultsMaxPoll   = 5 * time.Second
)

// vulnResultFetcher is the one call awaitVulnResult needs.
type vulnResultFetcher interface {
	GetUploadVulnResult(ctx context.Context, req *policy.GetUploadVulnResultReq) (*policy.GetUploadVulnResultResp, error)
}

// awaitVulnResult polls the upstream for the vulnerability result of one
// upload until it is decided or timeout runs out. It always returns an
// assessment, and a report only when the state is READY: every other outcome
// is "no data", which callers must never render as zero findings.
func awaitVulnResult(ctx context.Context, upstream vulnResultFetcher, req *policy.GetUploadVulnResultReq, timeout time.Duration) (*policy.VulnAssessment, *mvd.VulnReport) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	interval := vulnResultsFirstPoll
	var lastErr error
	for {
		resp, err := upstream.GetUploadVulnResult(ctx, req)
		if err == nil {
			switch resp.GetState() {
			case policy.VulnResultState_VULN_RESULT_STATE_READY:
				return &policy.VulnAssessment{
					State:         policy.VulnResultState_VULN_RESULT_STATE_READY,
					NewerScan:     resp.GetNewerScan(),
					NewerScanTime: resp.GetNewerScanTime(),
				}, resp.GetReport()
			case policy.VulnResultState_VULN_RESULT_STATE_NOT_ASSESSED,
				policy.VulnResultState_VULN_RESULT_STATE_FAILED:
				return &policy.VulnAssessment{State: resp.GetState(), Message: resp.GetMessage()}, nil
			}
			// PENDING, or a state this client doesn't know: ask again.
			lastErr = nil
		} else {
			switch status.Code(err) {
			case codes.NotFound, codes.Unimplemented:
				// The upstream predates the RPC (NotFound is ranger's
				// unknown-method answer), or there is no upstream that
				// analyses vulnerabilities.
				return &policy.VulnAssessment{
					State:   policy.VulnResultState_VULN_RESULT_STATE_NOT_ASSESSED,
					Message: "Mondoo Platform does not report vulnerability results for scans",
				}, nil
			}
			if ctx.Err() == nil {
				log.Debug().Err(err).Str("asset", req.GetScopeMrn()).Msg("could not get vulnerability results, retrying")
				lastErr = err
			}
		}

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return &policy.VulnAssessment{
					State:   policy.VulnResultState_VULN_RESULT_STATE_FAILED,
					Message: lastErr.Error(),
				}, nil
			}
			return &policy.VulnAssessment{
				State:   policy.VulnResultState_VULN_RESULT_STATE_PENDING,
				Message: fmt.Sprintf("Mondoo Platform did not finish analyzing vulnerabilities within %s", timeout),
			}, nil
		case <-time.After(interval):
		}
		interval = min(interval*2, vulnResultsMaxPoll)
	}
}
