// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package exceptions

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/cli/config"
)

var (
	now = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	ctx = &policy.ExceptionSource{Scope: ScopeContext, Provider: "terraform", Path: "/repo/mondoo.yml"}
)

func entry(checks []string, action string, paths ...string) config.Exception {
	return config.Exception{Checks: checks, Action: action, Justification: "because", Paths: paths}
}

func messages(issues []Issue) string {
	var res []string
	for _, i := range issues {
		res = append(res, i.String())
	}
	return strings.Join(res, "\n")
}

func TestParse(t *testing.T) {
	t.Run("valid entries", func(t *testing.T) {
		raw := []config.Exception{
			{Title: " t ", Checks: []string{"a", " b "}, Action: "Risk-Accepted", Justification: " j ", ValidUntil: "2026-11-01"},
			{Checks: []string{"c"}, Action: "disable", Justification: "j", Paths: []string{"./infra/staging/"}},
		}
		entries, issues := Parse(ctx, raw, now)
		require.Empty(t, issues, messages(issues))
		require.Len(t, entries, 2)

		assert.Equal(t, "t", entries[0].Title)
		assert.Equal(t, []string{"a", "b"}, entries[0].Checks)
		assert.Equal(t, policy.ExceptionAction_EXCEPTION_ACTION_RISK_ACCEPTED, entries[0].Action)
		assert.Equal(t, "j", entries[0].Justification)
		// a date is valid through the end of that day
		assert.Equal(t, time.Date(2026, 11, 1, 23, 59, 59, 0, time.UTC), entries[0].ValidUntil)

		assert.Equal(t, []string{"infra/staging"}, entries[1].Paths)
		assert.Equal(t, 2, entries[1].Index)
	})

	invalid := []struct {
		name string
		raw  config.Exception
		msg  string
	}{
		{"no checks", config.Exception{Action: "disable", Justification: "j"}, "names no checks"},
		{"glob check", entry([]string{"mondoo-*"}, "disable"), "globs are not supported"},
		{"no justification", config.Exception{Checks: []string{"a"}, Action: "disable", Justification: "  "}, "no justification"},
		{"no action", config.Exception{Checks: []string{"a"}, Justification: "j"}, "has no action"},
		{"unknown action", entry([]string{"a"}, "snooze"), `unknown action "snooze"`},
		{"out of scope", entry([]string{"a"}, "out-of-scope"), "statement about controls"},
		{"bad date", config.Exception{Checks: []string{"a"}, Action: "risk-accepted", Justification: "j", ValidUntil: "next week"}, "not an RFC3339"},
		{"absolute path", entry([]string{"a"}, "disable", "/etc"), "is absolute"},
		{"escaping path", entry([]string{"a"}, "disable", "infra/../../x"), "leaves the config's directory"},
		{"glob path", entry([]string{"a"}, "disable", "infra/*/staging"), "use a path prefix"},
		{"empty path", entry([]string{"a"}, "disable", " "), "empty path"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			entries, issues := Parse(ctx, []config.Exception{tc.raw}, now)
			assert.Empty(t, entries)
			require.NotEmpty(t, issues)
			assert.Equal(t, Error, issues[0].Severity)
			assert.Contains(t, messages(issues), tc.msg)
		})
	}

	t.Run("an invalid entry does not sink the file", func(t *testing.T) {
		entries, issues := Parse(ctx, []config.Exception{entry(nil, "disable"), entry([]string{"a"}, "disable")}, now)
		assert.Len(t, entries, 1)
		assert.Len(t, issues, 1)
	})

	t.Run("valid_until on disable warns and is dropped", func(t *testing.T) {
		raw := entry([]string{"a"}, "disable")
		raw.ValidUntil = "2027-01-01"
		entries, issues := Parse(ctx, []config.Exception{raw}, now)
		require.Len(t, entries, 1)
		assert.True(t, entries[0].ValidUntil.IsZero())
		require.Len(t, issues, 1)
		assert.Equal(t, Warning, issues[0].Severity)
	})

	t.Run("expired and expiring entries are kept and reported", func(t *testing.T) {
		expired := entry([]string{"a"}, "risk-accepted")
		expired.ValidUntil = "2026-01-01"
		expiring := entry([]string{"b"}, "risk-accepted")
		expiring.ValidUntil = "2026-10-10"
		entries, issues := Parse(ctx, []config.Exception{expired, expiring}, now)
		require.Len(t, entries, 2)
		assert.True(t, entries[0].Expired(now))
		assert.False(t, entries[1].Expired(now))
		assert.Contains(t, messages(issues), "expired on 2026-01-01")
		assert.Contains(t, messages(issues), "expires on 2026-10-10")
	})

	t.Run("paths are not available at user scope", func(t *testing.T) {
		user := &policy.ExceptionSource{Scope: ScopeUser, Path: "mondoo.yml"}
		entries, issues := Parse(user, []config.Exception{entry([]string{"a"}, "disable", "infra")}, now)
		assert.Empty(t, entries)
		assert.Contains(t, messages(issues), "paths only apply")
	})

	t.Run("conflicting actions at equal specificity", func(t *testing.T) {
		raw := []config.Exception{
			entry([]string{"a", "x"}, "disable", "infra"),
			entry([]string{"a"}, "risk-accepted", "infra"),
			entry([]string{"a"}, "risk-accepted", "infra/prod"), // more specific: fine
			entry([]string{"b"}, "risk-accepted"),
			entry([]string{"b"}, "risk-accepted"), // same action: fine
		}
		entries, issues := Parse(ctx, raw, now)
		assert.Contains(t, messages(issues), "exceptions 1 and 2 give check a different actions for path infra")
		require.Len(t, entries, 3)
		assert.Equal(t, 3, entries[0].Index)
	})
}

func TestMatchAsset(t *testing.T) {
	raw := []config.Exception{
		entry([]string{"logging"}, "risk-accepted"),
		entry([]string{"public"}, "risk-accepted", "infra/staging"),
		entry([]string{"public"}, "disable", "infra/staging/legacy"),
		entry([]string{"versioning"}, "risk-accepted", "infra"),
	}
	entries, issues := Parse(ctx, raw, now)
	require.Empty(t, issues, messages(issues))

	checks := func(ms []Match) map[string]string {
		res := map[string]string{}
		for _, m := range ms {
			res[m.Check] = ActionName(m.Entry.Action) + "@" + m.MatchedPath
		}
		return res
	}

	assert.Equal(t, map[string]string{"logging": "risk-accepted@"}, checks(MatchAsset(entries, "")),
		"an asset without a path is governed by unscoped entries only")
	assert.Equal(t, map[string]string{"logging": "risk-accepted@"}, checks(MatchAsset(entries, ".")))
	assert.Equal(t, map[string]string{
		"logging":    "risk-accepted@",
		"public":     "risk-accepted@infra/staging",
		"versioning": "risk-accepted@infra",
	}, checks(MatchAsset(entries, "infra/staging/web")))
	assert.Equal(t, map[string]string{
		"logging":    "risk-accepted@",
		"public":     "disable@infra/staging/legacy",
		"versioning": "risk-accepted@infra",
	}, checks(MatchAsset(entries, "infra/staging/legacy")), "the longer prefix wins")
	assert.Equal(t, map[string]string{
		"logging":    "risk-accepted@",
		"versioning": "risk-accepted@infra",
	}, checks(MatchAsset(entries, "infra/staging-2")), "prefixes match on directory boundaries")
}

func TestProtoAndGroups(t *testing.T) {
	raw := []config.Exception{
		{Title: "t", Checks: []string{"logging", "//policy.api.mondoo.app/queries/versioning", "gone"},
			Action: "risk-accepted", Justification: "j", ValidUntil: "2026-11-01"},
		entry([]string{"public"}, "disable", "infra"),
	}
	entries, issues := Parse(ctx, raw, now)
	require.Empty(t, issues)

	resolve := func(uid string) string {
		if uid == "gone" {
			return ""
		}
		return "//policy.api.mondoo.app/queries/" + uid
	}
	var protos []*policy.ExceptionEntry
	for _, m := range MatchAsset(entries, "infra/prod") {
		protos = append(protos, ToProto(m, resolve))
	}
	require.Len(t, protos, 4)

	assert.Equal(t, "logging", protos[0].CheckUid)
	assert.Equal(t, "//policy.api.mondoo.app/queries/logging", protos[0].CheckMrn)
	assert.Equal(t, entries[0].ValidUntil.Unix(), protos[0].ValidUntil.Seconds)
	assert.Equal(t, "", protos[1].CheckUid, "an MRN is used as written")
	assert.Equal(t, "//policy.api.mondoo.app/queries/versioning", protos[1].CheckMrn)
	assert.Equal(t, "", protos[2].CheckMrn, "an unknown check has no MRN")
	assert.Equal(t, "infra", protos[3].MatchedPath)

	groups := PolicyGroups(protos)
	require.Len(t, groups, 3, "the unknown check is left out")
	assert.Equal(t, policy.GroupType_IGNORED, groups[0].Type)
	assert.Equal(t, "j", groups[0].Docs.Justification)
	assert.Equal(t, entries[0].ValidUntil.Unix(), groups[0].Valid.Until.Seconds)
	assert.Equal(t, policy.GroupType_DISABLE, groups[2].Type)
	assert.Nil(t, groups[2].Valid)

	t.Run("keys and checksums", func(t *testing.T) {
		assert.Len(t, AllKeys(entries), 4)
		// keys do not depend on the order of entries or on their text
		reordered, _ := Parse(ctx, []config.Exception{raw[1], raw[0]}, now)
		assert.Equal(t, AllKeys(entries), AllKeys(reordered))
		assert.Equal(t, SetChecksum(entries), SetChecksum(reordered))

		changed := raw[0]
		changed.Justification = "other"
		edited, _ := Parse(ctx, []config.Exception{changed, raw[1]}, now)
		assert.Equal(t, AllKeys(entries), AllKeys(edited))
		assert.NotEqual(t, SetChecksum(entries), SetChecksum(edited))
	})
}
