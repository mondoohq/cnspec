// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"net/http"
	"os"

	"go.mondoo.com/mql/v13/cli/config"
)

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
func UploadFile(ctx context.Context, url string, headers map[string]string, filePath string, contentType string) (*http.Response, error) {
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

	client, err := newHTTPClient()
	if err != nil {
		return nil, err
	}
	return client.Do(req)
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
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = http.ProxyURL(proxy)
	return &http.Client{Transport: tr}, nil
}
