// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"go.mondoo.com/mql/v13/cli/config"
)

// Result carries the outcome of an upload attempt. BytesSent and Duration are
// populated even when err is non-nil, so a caller can tell a connection that
// never established (BytesSent == 0) from one that stalled mid-body
// (0 < BytesSent < ContentLength). The receiving end cannot make that
// distinction on its own — a stalled upload and one that never started look
// identical from there — which is why it is measured here.
type Result struct {
	// Response is nil when the request failed before a response was received.
	// The caller is responsible for closing its body.
	Response  *http.Response
	BytesSent int64
	Duration  time.Duration
}

// countingReader counts the bytes read out of it. http.Client reads the request
// body as it writes it to the wire, so the count is a lower bound on bytes that
// reached the network — good enough to classify a stall.
//
// The count is atomic because http.Client may read the body on a different
// goroutine than the one that returns from Do.
type countingReader struct {
	r io.Reader
	n atomic.Int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n.Add(int64(n))
	return n, err
}

// UploadFile uploads a file to a pre-signed URL via HTTP PUT.
//
// The request honors the Mondoo CLI's api_proxy setting in addition to the
// standard HTTPS_PROXY/HTTP_PROXY env vars: config.GetAPIProxy() resolves
// MONDOO_API_PROXY, viper's api_proxy (mondoo.yml / --api-proxy), and finally
// HTTPS_PROXY. When no proxy is configured, the default transport's
// http.ProxyFromEnvironment is used (which also honors NO_PROXY).
//
// It sets the provided headers and Content-Type to application/octet-stream.
// The caller is responsible for checking the response status code and closing
// the response body.
//
// The returned Result reports bytes sent and elapsed time on both the success
// and the error path — see Result.
func UploadFile(ctx context.Context, url string, headers map[string]string, filePath string, contentType string) (Result, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return Result{}, err
	}

	counter := &countingReader{r: file}
	req, err := http.NewRequestWithContext(ctx, "PUT", url, counter)
	if err != nil {
		return Result{}, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// This assignment is REQUIRED — do not drop it or move it above the request
	// construction. net/http only infers ContentLength for *bytes.Buffer,
	// *bytes.Reader and *strings.Reader; for anything else it leaves the length
	// unset and falls back to chunked transfer encoding, which signed-URL
	// endpoints commonly reject. countingReader also masks the concrete body
	// type, so nothing downstream can recover the size on its own.
	req.ContentLength = fileInfo.Size()
	req.Header.Set("Content-Type", contentType)

	client, err := newHTTPClient()
	if err != nil {
		return Result{}, err
	}

	start := time.Now()
	resp, err := client.Do(req)
	// BytesSent/Duration are filled in on both paths on purpose.
	return Result{Response: resp, BytesSent: counter.n.Load(), Duration: time.Since(start)}, err
}

// newHTTPClient builds the HTTP client used by UploadFile. When api_proxy is
// configured (via mondoo.yml, MONDOO_API_PROXY, --api-proxy, or HTTPS_PROXY)
// the transport is set to route through that proxy URL; otherwise we return
// a plain client whose default transport already honors HTTP(S)_PROXY/NO_PROXY
// via http.ProxyFromEnvironment.
func newHTTPClient() (*http.Client, error) {
	proxy, err := config.GetAPIProxy()
	if err != nil {
		return nil, err
	}
	if proxy == nil {
		return &http.Client{}, nil
	}
	// Clone the default transport when possible so we inherit TLS settings,
	// timeouts, and connection-pool tuning. Guard the assertion: callers (or
	// tests) may have replaced http.DefaultTransport with a non-*http.Transport
	// wrapper, in which case we fall back to a fresh transport rather than
	// panicking.
	var tr *http.Transport
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		tr = base.Clone()
	} else {
		tr = &http.Transport{}
	}
	tr.Proxy = http.ProxyURL(proxy)
	return &http.Client{Transport: tr}, nil
}
