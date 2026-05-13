// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"net/http"
	"os"
)

// UploadFile uploads a file to a pre-signed URL via HTTP PUT using the default
// HTTP client. The default client honors HTTP(S)_PROXY/NO_PROXY environment
// variables but does not consult the Mondoo CLI's api_proxy config; callers
// that need to honor api_proxy should use UploadFileWithClient.
//
// It sets the provided headers and Content-Type to application/octet-stream.
// The caller is responsible for checking the response status code and closing
// the response body.
func UploadFile(ctx context.Context, url string, headers map[string]string, filePath string, contentType string) (*http.Response, error) {
	return UploadFileWithClient(ctx, url, headers, filePath, contentType, nil)
}

// UploadFileWithClient is like UploadFile but routes the request through the
// supplied HTTP client. Pass a client configured with a proxy-aware transport
// to honor the Mondoo CLI's api_proxy setting. If client is nil, a default
// http.Client is used (env-var proxies only).
func UploadFileWithClient(ctx context.Context, url string, headers map[string]string, filePath string, contentType string, client *http.Client) (*http.Response, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	req, err := http.NewRequestWithContext(ctx, "PUT", url, file)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	req.ContentLength = fileInfo.Size()
	req.Header.Set("Content-Type", contentType)

	if client == nil {
		client = &http.Client{}
	}
	return client.Do(req)
}
