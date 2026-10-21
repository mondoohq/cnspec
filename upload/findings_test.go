// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

// fakeResolver stands in for *policy.PolicyResolverClient. It hands out the test
// server URL and records the completion call.
type fakeResolver struct {
	uploadURL        string
	sessionID        string
	getCalls         int
	completedSession string
	completedScope   string
	// completed records every completion call, in order; completeErr fails them.
	completed   []*policy.ReportUploadCompletedReq
	completeErr error
}

func (f *fakeResolver) GetUploadURL(_ context.Context, in *policy.GetUploadURLReq) (*policy.GetUploadURLResp, error) {
	f.getCalls++
	// The findings upload must request the third-party-findings kind.
	if in.Kind != policy.UploadURLKind_UPLOAD_URL_KIND_THIRD_PARTY_FINDINGS {
		return nil, assert.AnError
	}
	return &policy.GetUploadURLResp{
		UploadSessionId: f.sessionID,
		UploadUrl: &policy.UploadURL{
			Url:     f.uploadURL,
			Headers: map[string]string{"x-test": "1"},
		},
	}, nil
}

func (f *fakeResolver) ReportUploadCompleted(_ context.Context, in *policy.ReportUploadCompletedReq) (*policy.Empty, error) {
	f.completedSession = in.UploadSessionId
	f.completedScope = in.ScopeMrn
	f.completed = append(f.completed, in)
	if f.completeErr != nil {
		return nil, f.completeErr
	}
	return &policy.Empty{}, nil
}

func TestScoreAssets(t *testing.T) {
	const assetA = "//assets.api.mondoo.app/spaces/test/assets/a"
	const assetB = "//assets.api.mondoo.app/spaces/test/assets/b"

	t.Run("each asset is scored under the third-party session id", func(t *testing.T) {
		resolver := &fakeResolver{}
		scoreAssets(context.Background(), resolver, []string{assetA, assetB}, "VEX_SOURCE_XGREP")

		require.Len(t, resolver.completed, 2)
		for i, want := range []string{assetA, assetB} {
			// The session id the platform parses as "third-party upload of this
			// source, with Mondoo enrichment": no upload record behind it.
			assert.Equal(t, "UPLOAD_URL_KIND_THIRD_PARTY_FINDINGS-VEX_SOURCE_XGREP-true", resolver.completed[i].UploadSessionId)
			assert.Equal(t, want, resolver.completed[i].ScopeMrn)
		}
		assert.Zero(t, resolver.getCalls, "scoring requests no upload URL")
	})

	t.Run("no assets makes no call", func(t *testing.T) {
		resolver := &fakeResolver{}
		scoreAssets(context.Background(), resolver, nil, "VEX_SOURCE_XGREP")
		assert.Empty(t, resolver.completed)
	})

	t.Run("a failure does not stop the other assets", func(t *testing.T) {
		// A refusal is a PermissionDenied status, which is not retried.
		resolver := &fakeResolver{completeErr: status.Error(codes.PermissionDenied, "permission denied")}
		scoreAssets(context.Background(), resolver, []string{assetA, assetB}, "VEX_SOURCE_XGREP")
		require.Len(t, resolver.completed, 2)
		assert.Equal(t, assetB, resolver.completed[1].ScopeMrn)
	})
}

func TestUploadFindingsWithNothingToDo(t *testing.T) {
	// No documents and no assets to score: nothing happens, not even a
	// credential lookup, so a credential-less caller is not turned into an error.
	t.Setenv("MONDOO_CONFIG_PATH", t.TempDir()+"/missing.yml")
	require.NoError(t, UploadFindings(context.Background(), Opts{}, nil, "VEX_SOURCE_XGREP"))
}

func TestDoUpload(t *testing.T) {
	var (
		gotBody   []byte
		gotMethod string
		gotCType  string
		gotHeader string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCType = r.Header.Get("Content-Type")
		gotHeader = r.Header.Get("x-test")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resolver := &fakeResolver{uploadURL: srv.URL, sessionID: "sess-123"}

	docs := fex.FexToDocuments([]*fex.FindingExchange{
		{Id: "f1", Summary: "SQL injection", Status: fex.Status_STATUS_AFFECTED, Source: &fex.Source{Name: "xgrep"}},
	})
	req := &fex.FindingsUploadRequest{
		Findings:     docs,
		Source:       "xgrep",
		CreateAssets: true,
		SpaceMrn:     "//captain.api.mondoo.app/spaces/test",
	}
	data, err := protojson.Marshal(req)
	require.NoError(t, err)

	err = doUpload(context.Background(), resolver, srv.Client(), data, req.SpaceMrn)
	require.NoError(t, err)

	// The signed-URL PUT carried the protojson request, content type, and the
	// resolver-provided header.
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "application/json", gotCType)
	assert.Equal(t, "1", gotHeader)

	var roundtrip fex.FindingsUploadRequest
	require.NoError(t, protojson.Unmarshal(gotBody, &roundtrip))
	assert.Equal(t, "xgrep", roundtrip.Source)
	assert.True(t, roundtrip.CreateAssets)
	require.Len(t, roundtrip.Findings, 1)
	assert.Equal(t, "f1", roundtrip.Findings[0].GetFex().GetId())

	// Completion was signaled for the session whose PUT succeeded.
	assert.Equal(t, 1, resolver.getCalls)
	assert.Equal(t, "sess-123", resolver.completedSession)
	assert.Equal(t, req.SpaceMrn, resolver.completedScope)
}

func TestResolverHTTPClient(t *testing.T) {
	// nil falls back to a non-nil default client.
	assert.NotNil(t, resolverHTTPClient(nil))

	// a supplied client is used as-is.
	custom := &http.Client{}
	assert.Same(t, custom, resolverHTTPClient(custom))
}

func TestPutHTTPClient(t *testing.T) {
	t.Run("nil builds a default client with the upload timeout", func(t *testing.T) {
		c, err := putHTTPClient(nil)
		require.NoError(t, err)
		require.NotNil(t, c)
		assert.Equal(t, findingsUploadTimeout, c.Timeout)
	})

	t.Run("client without a timeout is cloned, caller left untouched", func(t *testing.T) {
		caller := &http.Client{Transport: http.DefaultTransport}
		c, err := putHTTPClient(caller)
		require.NoError(t, err)
		// A copy is returned with the upload timeout set...
		assert.NotSame(t, caller, c)
		assert.Equal(t, findingsUploadTimeout, c.Timeout)
		// ...and the same transport is preserved (so instrumentation survives).
		assert.Same(t, caller.Transport, c.Transport)
		// ...while the caller's client is unchanged.
		assert.Equal(t, time.Duration(0), caller.Timeout)
	})

	t.Run("client with its own timeout is used as-is", func(t *testing.T) {
		caller := &http.Client{Timeout: 30 * time.Second}
		c, err := putHTTPClient(caller)
		require.NoError(t, err)
		assert.Same(t, caller, c)
		assert.Equal(t, 30*time.Second, c.Timeout)
	})
}
