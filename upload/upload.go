// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"context"
	"net/http"
	"os"
)

// UploadFile uploads a file to a pre-signed URL via HTTP PUT.
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

	client := &http.Client{}
	return client.Do(req)
}
