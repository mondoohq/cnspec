package scan

import (
	"context"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
)

type fetcher struct {
	cache        map[string]*policy.Bundle
	github_token string
}

var reGithubToken = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

func newFetcher() *fetcher {
	github_token := os.Getenv("GITHUB_TOKEN")
	if !reGithubToken.MatchString(github_token) {
		log.Warn().Msg("invalid github token via environment variable, ignoring it")
		github_token = ""
	}

	return &fetcher{
		cache:        map[string]*policy.Bundle{},
		github_token: github_token,
	}
}

func (f *fetcher) fetchBundles(ctx context.Context, urls ...string) (*policy.Bundle, error) {
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
		if _, err := cur.Compile(ctx, nil); err != nil {
			return nil, errors.Wrap(err, "failed to compile fetched bundle")
		}

		if err = res.AddBundle(cur); err != nil {
			return nil, errors.Wrap(err, "failed to add fetched bundle")
		}
	}

	return res, nil
}

func (f *fetcher) fetchBundle(url string) (*policy.Bundle, error) {
	if f.github_token != "" {
		url += "?token=" + f.github_token
	}

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

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; cnquery/1.0; +http://www.mondoo.com)")

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
