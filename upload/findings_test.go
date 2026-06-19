// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
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
	return &policy.Empty{}, nil
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
