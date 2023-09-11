// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/policy"
)

type fetcher struct {
	cache map[string]*policy.Bundle
}

func newFetcher() *fetcher {
	return &fetcher{
		cache: map[string]*policy.Bundle{},
	}
}

func (f *fetcher) fetchBundles(ctx context.Context, schema llx.Schema, urls ...string) (*policy.Bundle, error) {
	var res *policy.Bundle = &policy.Bundle{}

	for i := range urls {
		url := urls[i]
		if cur, ok := f.cache[url]; ok {
			res.AddBundle(cur)
			continue
		}

		cur, err := f.fetchBundle(url)
		if err != nil {
			return nil, err
		}

		// need to generate MRNs for everything
		if _, err := cur.Compile(ctx, schema, nil); err != nil {
			return nil, errors.Wrap(err, "failed to compile fetched bundle")
		}

		if err = res.AddBundle(cur); err != nil {
			return nil, errors.Wrap(err, "failed to add fetched bundle")
		}
	}

	return res, nil
}

func (f *fetcher) fetchBundle(url string) (*policy.Bundle, error) {
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set up request to fetch bundle")
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; cnspec/"+cnspec.Version+"; +http://www.mondoo.com)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("failed to fetch policy bundle from " + url + ": " + resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return policy.BundleFromYAML(raw)
}
