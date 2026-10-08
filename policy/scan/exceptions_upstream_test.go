// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/cli/config"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
	"go.mondoo.com/mql/providers-sdk/v1/upstream"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

const upstreamQueries = "//policy.api.mondoo.app/queries/"

type stubHub struct {
	policy.PolicyHub
}

func (stubHub) GetBundle(context.Context, *policy.Mrn) (*policy.Bundle, error) {
	return &policy.Bundle{Queries: []*policy.Mquery{
		{Uid: "accepted-check", Mrn: upstreamQueries + "accepted-check"},
		{Uid: "pending-check", Mrn: upstreamQueries + "pending-check"},
		{Uid: "user-check", Mrn: upstreamQueries + "user-check"},
	}}, nil
}

type stubResolver struct {
	policy.PolicyResolver
	err  error
	reqs []*policy.SubmitExceptionsReq
}

// SubmitExceptions accepts pending-check for review and everything else
// without approval.
func (r *stubResolver) SubmitExceptions(_ context.Context, req *policy.SubmitExceptionsReq) (*policy.SubmitExceptionsResp, error) {
	r.reqs = append(r.reqs, req)
	if r.err != nil {
		return nil, r.err
	}
	resp := &policy.SubmitExceptionsResp{}
	for _, e := range req.Entries {
		outcome := policy.ExceptionOutcome_EXCEPTION_OUTCOME_AUTO_ACCEPTED
		if e.CheckUid == "pending-check" {
			outcome = policy.ExceptionOutcome_EXCEPTION_OUTCOME_PENDING
		}
		resp.Decisions = append(resp.Decisions, &policy.ExceptionDecision{
			Entry:   &policy.ExceptionEntry{Key: e.Key, Source: e.Source},
			Outcome: outcome,
		})
	}
	return resp, nil
}

func upstreamScanner(t *testing.T, resolver *stubResolver, submit bool) *localAssetScanner {
	t.Helper()
	_, services, err := inmemory.NewServices(testutils.Local())
	require.NoError(t, err)
	services.Upstream = &policy.Services{PolicyHub: stubHub{}, PolicyResolver: resolver}

	user := []config.Exception{{Checks: []string{"user-check", "accepted-check"}, Action: "workaround", Justification: "user"}}
	opts := &LocalScanner{}
	WithExceptions(user, "/home/u/.config/mondoo/mondoo.yml", submit)(opts)

	return &localAssetScanner{
		services:   services,
		exceptions: opts.exceptions,
		job: &AssetJob{
			Ctx:            context.Background(),
			UpstreamConfig: &upstream.UpstreamConfig{SpaceMrn: "//captain.api.mondoo.app/spaces/s"},
			Asset: &inventory.Asset{
				Mrn: "//assets.api.mondoo.app/spaces/s/assets/a",
				ContextConfig: &inventory.ContextConfig{
					Content: []byte(`
exceptions:
  - checks: [accepted-check, unknown-check]
    action: risk-accepted
    justification: accepted
  - checks: [pending-check]
    action: false-positive
    justification: wrong
  - checks: [elsewhere-check]
    action: disable
    justification: other module
    paths: [infra/prod]
`),
					Origin:    &inventory.ConfigOrigin{Provider: "github", Repository: "github.com/acme/infra", Ref: "main", Path: "mondoo.yml"},
					AssetPath: "infra/staging",
				},
			},
		},
	}
}

func byCheck(d *policy.ExceptionDecisions) map[string]*policy.ExceptionDecision {
	res := map[string]*policy.ExceptionDecision{}
	for _, item := range d.GetItems() {
		res[item.Entry.CheckUid] = item
	}
	return res
}

func TestApplyExceptionsWithUpstream(t *testing.T) {
	t.Run("authoritative run submits before resolution", func(t *testing.T) {
		resolver := &stubResolver{}
		s := upstreamScanner(t, resolver, true)

		decisions := byCheck(s.applyExceptions())
		assert.Nil(t, s.services.AssetExceptions, "an upstream-resolved policy is never adjusted locally")

		assert.Equal(t, policy.ExceptionOutcome_EXCEPTION_OUTCOME_AUTO_ACCEPTED, decisions["accepted-check"].Outcome)
		assert.Equal(t, policy.ExceptionOutcome_EXCEPTION_OUTCOME_PENDING, decisions["pending-check"].Outcome)
		assert.Equal(t, policy.ExceptionOutcome_EXCEPTION_OUTCOME_UNKNOWN_CHECK, decisions["unknown-check"].Outcome)
		assert.Equal(t, policy.ExceptionOutcome_EXCEPTION_OUTCOME_AUTO_ACCEPTED, decisions["user-check"].Outcome)
		assert.Equal(t, "context", decisions["accepted-check"].Entry.Source.Scope, "the context config wins over the user config")
		assert.NotContains(t, decisions, "elsewhere-check")

		require.Len(t, resolver.reqs, 2)
		ctxReq := resolver.reqs[0]
		assert.Equal(t, "//assets.api.mondoo.app/spaces/s/assets/a", ctxReq.ScopeMrn)
		require.Len(t, ctxReq.Entries, 2, "the unknown check is not sent")
		assert.Equal(t, upstreamQueries+"accepted-check", ctxReq.Entries[0].CheckMrn)
		require.Len(t, ctxReq.Sources, 1)
		assert.Len(t, ctxReq.Sources[0].Keys, 4, "the declaration lists every entry of the file, not only the matched ones")
		assert.NotEmpty(t, ctxReq.Sources[0].Checksum)

		userReq := resolver.reqs[1]
		assert.Equal(t, "//captain.api.mondoo.app/spaces/s", userReq.ScopeMrn)
		assert.Len(t, userReq.Entries, 2, "user-scope entries go to the space in full")
		assert.Equal(t, "user", userReq.Sources[0].Source.Scope)

		// a second asset in the same run does not submit the user scope again
		s2 := upstreamScanner(t, resolver, true)
		s2.exceptions = s.exceptions
		s2.applyExceptions()
		assert.Len(t, resolver.reqs, 3)
	})

	t.Run("other runs do not submit", func(t *testing.T) {
		resolver := &stubResolver{}
		decisions := byCheck(upstreamScanner(t, resolver, false).applyExceptions())
		assert.Empty(t, resolver.reqs)
		assert.Equal(t, policy.ExceptionOutcome_EXCEPTION_OUTCOME_NOT_SUBMITTED, decisions["accepted-check"].Outcome)
		assert.Contains(t, decisions["accepted-check"].Reason, "default branch")
	})

	t.Run("an upstream without the RPC", func(t *testing.T) {
		resolver := &stubResolver{err: status.Error(codes.NotFound, "unknown method")}
		decisions := byCheck(upstreamScanner(t, resolver, true).applyExceptions())
		assert.Equal(t, policy.ExceptionOutcome_EXCEPTION_OUTCOME_NOT_SUBMITTED, decisions["accepted-check"].Outcome)
		assert.Equal(t, "not submitted: the upstream does not take exceptions", decisions["accepted-check"].Reason)
	})
}
