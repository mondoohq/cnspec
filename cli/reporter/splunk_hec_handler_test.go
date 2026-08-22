// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// splunkTestServer stands in for a HEC endpoint and records what it received.
func splunkTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, string) {
	t.Helper()
	// httptest listens on the loopback interface, which must not be proxied.
	t.Setenv("NO_PROXY", "127.0.0.1,localhost")
	t.Setenv("no_proxy", "127.0.0.1,localhost")

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, srv.URL + "/services/collector"
}

func hecHandler(t *testing.T, url string) *splunkHecHandler {
	t.Helper()
	conf := defaultPrintConfig()
	conf.format = FormatOcsfJson
	conf.ocsfFindings = OcsfFindingsDetection

	handler, err := newSplunkHecHandler(url, conf)
	require.NoError(t, err)
	// the backoff behavior is asserted by the retry counts, not by the wall clock
	handler.backoff = time.Millisecond
	return handler
}

func TestSplunkHecTargetDetection(t *testing.T) {
	for _, target := range []string{
		"https://splunk.example.com:8088/services/collector",
		"http://127.0.0.1:8088/services/collector/event",
		"https://splunk.example.com:8088/services/collector/",
	} {
		assert.Equal(t, SPLUNK_HEC, determineOutputType(target), target)
	}

	for _, target := range []string{
		"/tmp/report.jsonl",
		"https://example.com/not/a/collector",
		"",
	} {
		assert.NotEqual(t, SPLUNK_HEC, determineOutputType(target), target)
	}
}

func TestSplunkHecSendsEnvelopedEvents(t *testing.T) {
	var body atomic.Value
	var auth atomic.Value
	srv, url := splunkTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body.Store(string(raw))
		auth.Store(r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"text":"Success","code":0}`))
	})
	defer srv.Close()

	t.Setenv(splunkTokenEnv, "test-token")
	t.Setenv(splunkIndexEnv, "security")

	require.NoError(t, hecHandler(t, url).WriteReport(t.Context(), advisoryReportCollection()))

	assert.Equal(t, "Splunk test-token", auth.Load())

	lines := strings.Split(strings.TrimSpace(body.Load().(string)), "\n")
	require.NotEmpty(t, lines)

	sourcetypes := map[string]int{}
	for _, line := range lines {
		var envelope struct {
			Time       float64         `json:"time"`
			Host       string          `json:"host"`
			Source     string          `json:"source"`
			Sourcetype string          `json:"sourcetype"`
			Index      string          `json:"index"`
			Event      json.RawMessage `json:"event"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &envelope))

		sourcetypes[envelope.Sourcetype]++
		assert.Equal(t, "cnspec", envelope.Source)
		assert.Equal(t, "security", envelope.Index)
		assert.Equal(t, "X1", envelope.Host, "the asset becomes the Splunk host")
		assert.Greater(t, envelope.Time, 1.0e9, "HEC wants epoch seconds, not milliseconds")
		assert.Less(t, envelope.Time, 1.0e11)

		var event map[string]any
		require.NoError(t, json.Unmarshal(envelope.Event, &event))
		assert.NotZero(t, event["class_uid"], "the OCSF event travels whole inside the envelope")
	}

	// the sourcetypes the OCSF-CIM Add-On for Splunk keys off
	assert.NotContains(t, sourcetypes, "ocsf:compliance_finding",
		"detection mode reports checks as class 2004 only")
	assert.Equal(t, 3, sourcetypes["ocsf:detection_finding"])
	assert.Equal(t, 1, sourcetypes["ocsf:vulnerability_finding"])
	assert.Equal(t, 1, sourcetypes["ocsf:inventory_info"])
}

func TestSplunkHecRetriesServerErrors(t *testing.T) {
	var attempts atomic.Int32
	srv, url := splunkTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"text":"Server is busy","code":9}`))
			return
		}
		_, _ = w.Write([]byte(`{"text":"Success","code":0}`))
	})
	defer srv.Close()

	t.Setenv(splunkTokenEnv, "test-token")

	require.NoError(t, hecHandler(t, url).WriteReport(t.Context(), sampleReportCollection()))
	assert.EqualValues(t, 3, attempts.Load(), "a busy Splunk is retried until it accepts")
}

func TestSplunkHecDoesNotRetryRejections(t *testing.T) {
	var attempts atomic.Int32
	srv, url := splunkTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"text":"Invalid token","code":4}`))
	})
	defer srv.Close()

	t.Setenv(splunkTokenEnv, "wrong-token")

	err := hecHandler(t, url).WriteReport(t.Context(), sampleReportCollection())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid token")
	assert.EqualValues(t, 1, attempts.Load(), "a bad token is not worth retrying")
}

func TestSplunkHecRequiresToken(t *testing.T) {
	t.Setenv(splunkTokenEnv, "")

	err := hecHandler(t, "https://splunk.example.com:8088/services/collector").
		WriteReport(t.Context(), sampleReportCollection())
	require.Error(t, err)
	assert.Contains(t, err.Error(), splunkTokenEnv)
}

func TestSplunkHecRequiresOcsfFormat(t *testing.T) {
	conf := defaultPrintConfig()
	conf.format = FormatJSONv2

	_, err := newSplunkHecHandler("https://splunk.example.com:8088/services/collector", conf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ocsf-json")
}
