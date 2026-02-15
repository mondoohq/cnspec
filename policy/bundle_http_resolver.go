// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v12"
)

// HTTPDoer is an interface for making HTTP requests. It is implemented by
// *http.Client and can be replaced in tests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type httpBundleResolver struct {
	client HTTPDoer
}

func defaultHTTPBundleResolver() *httpBundleResolver {
	return &httpBundleResolver{}
}

// NewHTTPBundleResolver creates an HTTP bundle resolver with the provided HTTP
// client. This is useful for testing with httptest.NewServer.
func NewHTTPBundleResolver(client HTTPDoer) *httpBundleResolver {
	return &httpBundleResolver{client: client}
}

func (r *httpBundleResolver) IsApplicable(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

func (r *httpBundleResolver) Load(ctx context.Context, path string) (*Bundle, error) {
	client := r.client
	if client == nil {
		client = &http.Client{
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				r.URL.Opaque = r.URL.Path
				return nil
			},
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request for policy bundle")
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; cnspec/"+cnspec.Version+"; +http://www.mondoo.com)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch policy bundle from "+path)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, httpError(path, resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read policy bundle response from "+path)
	}

	log.Debug().Str("url", path).Msg("http>loaded bundle file from URL")
	return BundleFromYAML(data)
}

func httpError(url string, statusCode int, status string) error {
	switch statusCode {
	case http.StatusBadRequest:
		return errors.Newf("failed to fetch policy bundle from %s: the server indicated the request was malformed (HTTP 400 Bad Request)", url)
	case http.StatusForbidden:
		return errors.Newf("failed to fetch policy bundle from %s: access denied, ensure the URL is correct and publicly accessible or that valid credentials are provided (HTTP 403 Forbidden)", url)
	case http.StatusNotFound:
		return errors.Newf("failed to fetch policy bundle from %s: resource not found (HTTP 404 Not Found)", url)
	default:
		return errors.Newf("failed to fetch policy bundle from %s: %s", url, status)
	}
}
