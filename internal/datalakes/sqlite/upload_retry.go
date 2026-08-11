// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package sqlite

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/cnspec/v13/policy/scanstats"
	"go.mondoo.com/cnspec/v13/upload"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"google.golang.org/protobuf/types/known/anypb"
)

// uploadResolver is the slice of the upstream policy service the scan-database
// upload needs. *policy.Services satisfies it; a fake stands in for tests.
//
// upload/findings.go declares an equivalent interface, but it is unexported and
// in a different package, so this is a deliberate duplicate rather than a shared
// type.
type uploadResolver interface {
	GetUploadURL(ctx context.Context, in *policy.GetUploadURLReq) (*policy.GetUploadURLResp, error)
	ReportUploadCompleted(ctx context.Context, in *policy.ReportUploadCompletedReq) (*policy.Empty, error)
}

// scanUploadOutcome is what doScanUpload observed, so the caller can decide what
// to report without doScanUpload touching the health package. Keeping telemetry
// at the edge is what makes the retry logic testable without a global.
type scanUploadOutcome struct {
	sessionID string
	// attempts is the number of attempts actually made, not the budget. 1 on a
	// permanent error that failed fast; DefaultRetryAttempts when the budget was
	// spent.
	attempts int
	// last* describe the most recent FAILED attempt, and stay set even when a
	// later attempt succeeds — that is what a recovery report describes.
	lastKind   upload.FailureKind
	lastStatus int
	lastRes    upload.Result
	lastBody   string
	lastErrMsg string
	// okRes is the successful attempt, used for the upload stats.
	okRes      upload.Result
	bytesTotal int64
}

// recovered reports whether the upload succeeded only after failing at least
// once. Callers use it to pick between the failure and recovery reports.
func (o scanUploadOutcome) recovered() bool { return o.attempts > 1 }

// doScanUpload runs the signed-URL upload handshake with bounded retries:
// request a URL, PUT the scan database, then report completion.
//
// GetUploadURL and the PUT share one retry block because signed URLs expire
// quickly (X-Goog-Expires=299) — reusing a URL across a backoff window would PUT
// against an expired URL and 403. A fresh URL per attempt avoids that, and the
// PUT is idempotent (same object key, same content) so repeating it is safe.
// This mirrors upload/findings.go doUpload.
func doScanUpload(ctx context.Context, resolver uploadResolver, scanDataPath, assetMrn string, stats *scanstats.Collector) (scanUploadOutcome, error) {
	out := scanUploadOutcome{bytesTotal: fileSizeBytes(scanDataPath)}

	if err := upstream.WithRetry(ctx, "upload scan data", func() (bool, time.Duration, error) {
		out.attempts++

		urlResp, err := resolver.GetUploadURL(ctx, &policy.GetUploadURLReq{
			Kind:     policy.UploadURLKind_UPLOAD_URL_KIND_SCAN_DATABASE_V0,
			ScopeMrn: assetMrn,
		})
		if err != nil {
			out.lastKind = upload.FailureURLRequest
			out.lastErrMsg = "upload failed: could not get upload URL: " + upload.RedactError(err)
			// Unauthenticated and PermissionDenied are permanent: fail fast
			// rather than spending the budget re-asking the same question.
			return upstream.RetryableRPCError(err), 0, err
		}
		out.sessionID = urlResp.UploadSessionId

		uploadURL := urlResp.UploadUrl
		if uploadURL == nil {
			out.lastKind = upload.FailureURLRequest
			out.lastErrMsg = "upload failed: no upload URL for scan data store"
			// A server that returns no URL is broken, not busy.
			return false, 0, errors.New("no upload URL for scan data store")
		}

		res, err := upload.UploadFile(ctx, uploadURL.Url, uploadURL.Headers, scanDataPath, "application/octet-stream")
		if err != nil {
			out.lastKind = upload.ClassifyFailure(err)
			out.lastStatus = 0
			out.lastRes = res
			out.lastErrMsg = "upload failed: " + upload.RedactError(err)
			// Transport failure — the connection_reset/wsasend/dial-timeout
			// class. Always worth another attempt.
			return true, 0, err
		}
		defer res.Response.Body.Close()

		if res.Response.StatusCode != http.StatusOK && res.Response.StatusCode != http.StatusCreated {
			// Truncate to 512 bytes to avoid leaking sensitive details.
			body, _ := io.ReadAll(io.LimitReader(res.Response.Body, 512))
			out.lastKind = upload.FailureHTTPStatus
			out.lastStatus = res.Response.StatusCode
			out.lastRes = res
			out.lastBody = string(body)
			out.lastErrMsg = fmt.Sprintf("upload failed with status %d", res.Response.StatusCode)
			return upstream.RetryableHTTPStatus(res.Response.StatusCode),
				upstream.RetryAfter(res.Response.Header),
				fmt.Errorf("upload failed with status %d", res.Response.StatusCode)
		}

		out.okRes = res
		return false, 0, nil
	}); err != nil {
		return out, err
	}

	// Success-path baseline: without these, a slow failing upload cannot be
	// compared against the normal distribution. Records the successful attempt,
	// not the sum across retries.
	if stats != nil {
		stats.AddDuration(scanstats.MetricUploadDuration, out.okRes.Duration)
		if secs := out.okRes.Duration.Seconds(); secs > 0 {
			stats.AddDouble(scanstats.MetricUploadThroughput, "bps", float64(out.okRes.BytesSent*8)/secs)
		}
	}

	req := &policy.ReportUploadCompletedReq{
		UploadSessionId: out.sessionID,
		ScopeMrn:        assetMrn,
	}
	if stats != nil {
		if s := stats.ToProto(); s != nil {
			if details, aerr := anypb.New(s); aerr != nil {
				log.Warn().Err(aerr).Msg("failed to encode scan statistics; sending upload confirmation without them")
			} else {
				req.Details = details
			}
		}
	}

	// The PUT already succeeded, so the object IS in the bucket. Losing the
	// completion call means the platform never learns the data arrived and
	// discards it anyway, so this is worth retrying.
	if err := upstream.RetryRPC(ctx, "report upload completed", func() error {
		_, err := resolver.ReportUploadCompleted(ctx, req)
		return err
	}); err != nil {
		out.lastKind = upload.FailureReportRPC
		out.lastErrMsg = "upload failed: could not report completion: " + upload.RedactError(err)
		return out, err
	}

	return out, nil
}

// uploadReportKind selects which client report an outcome becomes. The values
// are read by the platform's error-report handler to pick the record type, so
// they are part of the wire contract — do not change them casually.
type uploadReportKind string

const (
	uploadReportKindFailure   uploadReportKind = "upload_failure"
	uploadReportKindRecovered uploadReportKind = "upload_recovered"
)

// uploadOutcomeTags builds the health.SendError tags for an upload outcome.
// Both report kinds carry the same fields — a recovery describes the last
// failure before success, so "what was it fighting" survives either way.
func uploadOutcomeTags(kind uploadReportKind, out scanUploadOutcome, assetMrn string) map[string]string {
	tags := uploadFailureTags(out.sessionID, assetMrn, out.lastKind, out.lastStatus, out.lastRes, out.bytesTotal)
	tags["reportKind"] = string(kind)
	tags["attempts"] = strconv.Itoa(out.attempts)
	if out.lastBody != "" {
		tags["responseBody"] = out.lastBody
	}
	return tags
}
