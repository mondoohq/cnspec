// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13/policy"
	cliconfig "go.mondoo.com/mql/v13/cli/config"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
	ranger "go.mondoo.com/ranger-rpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// findingsUploadTimeout bounds a single signed-URL PUT so a stalled connection
// can't hang indefinitely (the proxy-aware client has no timeout of its own).
// Retry/backoff of the upload operation itself is handled by upstream.WithRetry.
const findingsUploadTimeout = 2 * time.Minute

// ErrNoCredentials is returned when no service account config can be found. Use
// IsNoCredentials to let callers degrade to a local-only run.
var ErrNoCredentials = errors.New("no Mondoo service account credentials found")

// IsNoCredentials reports whether the error indicates missing credentials.
func IsNoCredentials(err error) bool {
	return errors.Is(err, ErrNoCredentials)
}

// Opts selects the Mondoo service-account config and target scope for a findings
// upload. ConfigPath is optional — when empty the standard config resolution
// (MONDOO_CONFIG_PATH, MONDOO_CONFIG_BASE64, AWS SSM, ~/.config/mondoo/mondoo.yml)
// applies. ScopeMrn overrides the scope from the config when set.
type Opts struct {
	ConfigPath string
	ScopeMrn   string

	// HTTPClient, when set, is used for both the resolver RPCs and the
	// signed-URL PUT, letting callers supply an instrumented client (tracing,
	// metrics, custom transport). When nil, default clients are used. The PUT
	// enforces findingsUploadTimeout: if the supplied client has no timeout, a
	// shallow copy is used for the PUT so the caller's client is left untouched.
	HTTPClient *http.Client
}

// UploadFindings uploads FEX/VEX documents to Mondoo Platform as third-party
// findings: it resolves credentials, requests a signed object-storage URL via
// the PolicyResolver, PUTs the protojson-encoded request, then signals
// completion. Each step is retried on transient (network/5xx/429) failures so a
// blip doesn't discard the collected scan data. source identifies the producing
// tool (e.g. "xgrep"). Returns ErrNoCredentials when no config is found (check
// with IsNoCredentials).
func UploadFindings(ctx context.Context, opts Opts, docs []*fex.FindingDocument, source string) error {
	if len(docs) == 0 {
		log.Debug().Msg("no findings to upload")
		return nil
	}

	creds, spaceMrn, err := LoadCredentials(opts)
	if err != nil {
		return err
	}

	plugin, err := upstream.NewServiceAccountRangerPlugin(creds)
	if err != nil {
		return fmt.Errorf("create auth plugin: %w", err)
	}

	resolver, err := policy.NewPolicyResolverClient(creds.ApiEndpoint, resolverHTTPClient(opts.HTTPClient), plugin)
	if err != nil {
		return fmt.Errorf("create policy resolver client: %w", err)
	}

	data, err := protojson.Marshal(&fex.FindingsUploadRequest{
		Findings:         docs,
		Source:           source,
		MondooEnrichment: true,
		SpaceMrn:         spaceMrn,
		CreateAssets:     true,
		ImportStartedAt:  timestamppb.Now(),
	})
	if err != nil {
		return fmt.Errorf("marshal findings: %w", err)
	}

	httpClient, err := putHTTPClient(opts.HTTPClient)
	if err != nil {
		return fmt.Errorf("create http client: %w", err)
	}

	if err := doUpload(ctx, resolver, httpClient, data, spaceMrn); err != nil {
		return err
	}

	log.Info().Int("findings", len(docs)).Str("source", source).Msg("uploaded findings to Mondoo Platform")
	return nil
}

// resolverHTTPClient returns the client used for the resolver RPCs: the
// caller-supplied client when set, otherwise ranger's default.
func resolverHTTPClient(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return ranger.DefaultHttpClient()
}

// putHTTPClient returns the client used for the signed-URL PUT, bounded by
// findingsUploadTimeout so a stalled connection can't hang indefinitely. When
// the caller supplies a client with no timeout of its own, a shallow copy is
// returned (the timeout is set on the copy, not the caller's shared client).
func putHTTPClient(c *http.Client) (*http.Client, error) {
	if c == nil {
		nc, err := newHTTPClient()
		if err != nil {
			return nil, err
		}
		nc.Timeout = findingsUploadTimeout
		return nc, nil
	}
	if c.Timeout == 0 {
		clone := *c
		clone.Timeout = findingsUploadTimeout
		return &clone, nil
	}
	return c, nil
}

// uploadResolver is the slice of the PolicyResolver client the findings upload
// needs. *policy.PolicyResolverClient satisfies it; a fake stands in for tests.
type uploadResolver interface {
	GetUploadURL(ctx context.Context, in *policy.GetUploadURLReq) (*policy.GetUploadURLResp, error)
	ReportUploadCompleted(ctx context.Context, in *policy.ReportUploadCompletedReq) (*policy.Empty, error)
}

// doUpload runs the signed-URL upload handshake: request a URL, PUT the payload,
// then report completion. Split out from UploadFindings so it can be tested
// against a fake resolver and an httptest server without real credentials.
func doUpload(ctx context.Context, resolver uploadResolver, httpClient *http.Client, data []byte, spaceMrn string) error {
	// A fresh signed URL is fetched per attempt (signed URLs expire quickly, so
	// reusing one across a backoff window would PUT against an expired URL and
	// 403), so GetUploadURL and the PUT live in the same retry block. The PUT to
	// object storage is idempotent (same object), so a retry is safe.
	// uploadSessionID is captured from the attempt whose PUT succeeds and used for
	// ReportUploadCompleted below.
	var uploadSessionID string
	if err := upstream.WithRetry(ctx, "upload findings", func() (bool, time.Duration, error) {
		resp, err := resolver.GetUploadURL(ctx, &policy.GetUploadURLReq{
			Kind:     policy.UploadURLKind_UPLOAD_URL_KIND_THIRD_PARTY_FINDINGS,
			ScopeMrn: spaceMrn,
		})
		if err != nil {
			return upstream.RetryableRPCError(err), 0, fmt.Errorf("get upload URL: %w", err)
		}
		uploadURL := resp.GetUploadUrl()

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL.GetUrl(), bytes.NewReader(data))
		if err != nil {
			return false, 0, err
		}
		req.ContentLength = int64(len(data))
		req.Header.Set("Content-Type", "application/json")
		for k, v := range uploadURL.GetHeaders() {
			req.Header.Set(k, v)
		}
		httpResp, err := httpClient.Do(req)
		if err != nil {
			return true, 0, err // network error: retry
		}
		defer func() { _ = httpResp.Body.Close() }()
		if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
			return upstream.RetryableHTTPStatus(httpResp.StatusCode), upstream.RetryAfter(httpResp.Header),
				fmt.Errorf("upload failed with status %d: %s", httpResp.StatusCode, string(body))
		}
		uploadSessionID = resp.GetUploadSessionId()
		return false, 0, nil
	}); err != nil {
		return err
	}

	return upstream.RetryRPC(ctx, "signal upload completed", func() error {
		_, err := resolver.ReportUploadCompleted(ctx, &policy.ReportUploadCompletedReq{
			UploadSessionId: uploadSessionID,
			ScopeMrn:        spaceMrn,
		})
		return err
	})
}

// LoadCredentials resolves the service-account credentials and scope MRN from
// opts, for callers that drive their own Mondoo RPC with the same auth as the
// findings upload. Returns ErrNoCredentials when no config is found (check with
// IsNoCredentials).
func LoadCredentials(opts Opts) (*upstream.ServiceAccountCredentials, string, error) {
	creds, err := loadServiceAccount(opts.ConfigPath)
	if err != nil {
		return nil, "", err
	}

	spaceMrn := opts.ScopeMrn
	if spaceMrn == "" {
		spaceMrn = creds.GetScopeMrn()
	}
	if spaceMrn == "" {
		spaceMrn = creds.GetParentMrn()
	}
	if spaceMrn == "" {
		return nil, "", fmt.Errorf("scope MRN is required (use --scope-mrn or set scope_mrn in config)")
	}

	return creds, spaceMrn, nil
}

// ValidateCredentials checks connectivity to Mondoo Platform with the given
// credentials, so a caller can fail fast before collecting findings.
func ValidateCredentials(ctx context.Context, creds *upstream.ServiceAccountCredentials) error {
	plugin, err := upstream.NewServiceAccountRangerPlugin(creds)
	if err != nil {
		return err
	}

	client, err := upstream.NewAgentManagerClient(creds.ApiEndpoint, ranger.DefaultHttpClient(), plugin)
	if err != nil {
		return err
	}

	_, err = client.PingPong(ctx, &upstream.Ping{})
	return err
}

// configMu serializes access to cli/config's process-global state. InitViperConfig
// reads a package-level path (UserProvidedPath) and populates the global viper
// instance, so concurrent callers would otherwise race on that shared state.
var configMu sync.Mutex

// loadServiceAccount resolves Mondoo service account credentials via mql's
// canonical config loader (cli/config). InitViperConfig handles every config
// source the Mondoo CLI supports — the explicit path, MONDOO_CONFIG_PATH,
// MONDOO_CONFIG_BASE64, AWS SSM Parameter Store, and autodetection of the
// default ~/.config/mondoo/mondoo.yml — and GetServiceCredential parses the
// loaded config into the upstream credential type (including SSH/WIF token
// exchange). Returns ErrNoCredentials when no credentials are configured.
func loadServiceAccount(configPath string) (*upstream.ServiceAccountCredentials, error) {
	cfg, err := readMondooConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("read mondoo config: %w", err)
	}

	creds := cfg.GetServiceCredential()
	if creds == nil {
		return nil, ErrNoCredentials
	}
	if creds.Mrn == "" {
		return nil, fmt.Errorf("invalid service account: missing mrn")
	}

	return creds, nil
}

// readMondooConfig points cli/config at configPath, (re)loads the global viper
// instance, and reads it back. The whole sequence holds configMu so a concurrent
// caller can't repopulate the global viper between InitViperConfig and Read and
// hand us the wrong config; defer also keeps the lock held if Read panics.
func readMondooConfig(configPath string) (*cliconfig.Config, error) {
	configMu.Lock()
	defer configMu.Unlock()
	cliconfig.UserProvidedPath = configPath
	cliconfig.InitViperConfig()
	return cliconfig.Read()
}
