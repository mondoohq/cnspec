// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/exceptions"
	"go.mondoo.com/mql/cli/config"
	"go.mondoo.com/mql/mrn"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// exceptionsConfig is how a scanner treats exceptions from config files
// (ADR-0006).
type exceptionsConfig struct {
	// user are the entries of the client config, which govern every asset.
	user []*exceptions.Entry
	// submit says whether this run may submit exceptions upstream. Only an
	// authoritative run does, so that a scan of a feature branch never turns
	// a developer's experiment into a shared exception.
	submit bool

	// userOnce submits the user-scope entries once per run, against the space;
	// they apply to every asset in it, not to one.
	userOnce      sync.Once
	userDecisions map[string]*policy.ExceptionDecision
}

// WithExceptions sets the user-scope exceptions, read from the client config
// at configPath, and whether this run submits exceptions upstream.
func WithExceptions(user []config.Exception, configPath string, submit bool) ScannerOption {
	return func(s *LocalScanner) {
		src := &policy.ExceptionSource{Scope: exceptions.ScopeUser, Path: configPath}
		entries, issues := exceptions.Parse(src, user, time.Now())
		logExceptionIssues(issues)
		s.exceptions = &exceptionsConfig{user: entries, submit: submit}
	}
}

func logExceptionIssues(issues []exceptions.Issue) {
	for _, i := range issues {
		if i.Severity == exceptions.Error {
			log.Warn().Msg("ignoring exception: " + i.String())
		} else {
			log.Warn().Msg(i.String())
		}
	}
}

// matchedException is one check exception that governs the asset being
// scanned, with every entry of the source it came from, which an upstream
// needs to know the source's complete set.
type matchedException struct {
	match  exceptions.Match
	source []*exceptions.Entry
}

// assetExceptions collects the exceptions that govern the asset: those of the
// context config the provider found at the scanned root, then those of the
// client config for checks the context config does not name. The context
// config is the narrower scope, closer to the thing being scanned, so it wins.
func (s *localAssetScanner) assetExceptions(now time.Time) (contextMatches []matchedException, userMatches []matchedException) {
	covered := map[string]struct{}{}

	if cc := s.job.Asset.GetContextConfig(); cc != nil && len(cc.Content) != 0 {
		src := exceptions.SourceFromOrigin(cc.Origin)
		parsed, err := config.ParseContextConfig(cc.Content)
		if err != nil {
			log.Warn().Err(err).Str("file", exceptions.SourceName(src)).Msg("ignoring exceptions in config at scanned root")
		} else {
			entries, issues := exceptions.Parse(src, parsed.Exceptions, now)
			logExceptionIssues(issues)
			for _, m := range exceptions.MatchAsset(entries, cc.AssetPath) {
				contextMatches = append(contextMatches, matchedException{match: m, source: entries})
				covered[m.Check] = struct{}{}
			}
		}
	}

	if s.exceptions != nil {
		for _, m := range exceptions.MatchAsset(s.exceptions.user, "") {
			if _, ok := covered[m.Check]; ok {
				continue
			}
			userMatches = append(userMatches, matchedException{match: m, source: s.exceptions.user})
		}
	}
	return contextMatches, userMatches
}

// checkResolver translates check UIDs to MRNs from the queries of a bundle.
func checkResolver(queries []*policy.Mquery) exceptions.CheckResolver {
	byUid := make(map[string]string, len(queries))
	for _, q := range queries {
		if q.Mrn == "" {
			continue
		}
		uid := q.Uid
		if uid == "" {
			uid, _ = mrn.GetResource(q.Mrn, policy.MRN_RESOURCE_QUERY)
		}
		if uid != "" {
			byUid[uid] = q.Mrn
		}
	}
	known := make(map[string]struct{}, len(byUid))
	for _, m := range byUid {
		known[m] = struct{}{}
	}
	return func(check string) string {
		if m, ok := byUid[check]; ok {
			return m
		}
		if _, ok := known[check]; ok {
			return check
		}
		return ""
	}
}

// applyExceptions decides what becomes of each exception that governs the
// asset, before its policy is resolved. Without an upstream cnspec applies them
// itself; with one it submits them and the upstream decides, and the resolved
// policy it returns reflects what it accepted. Either way every exception is
// accounted for in the returned decisions.
func (s *localAssetScanner) applyExceptions() *policy.ExceptionDecisions {
	now := time.Now()
	ctxMatches, userMatches := s.assetExceptions(now)
	if len(ctxMatches) == 0 && len(userMatches) == 0 {
		return nil
	}

	local := s.services.Upstream == nil || s.services.Incognito
	resolve := s.checkResolver(local)

	toProto := func(ms []matchedException) []*policy.ExceptionEntry {
		res := make([]*policy.ExceptionEntry, len(ms))
		for i := range ms {
			res[i] = exceptions.ToProto(ms[i].match, resolve)
		}
		return res
	}
	ctxEntries := toProto(ctxMatches)
	userEntries := toProto(userMatches)

	var decisions []*policy.ExceptionDecision
	if local {
		decisions = s.applyLocally(append(ctxEntries, userEntries...), now)
	} else {
		decisions = append(decisions, s.submitContext(ctxMatches, ctxEntries, now)...)
		decisions = append(decisions, s.submitUser(userEntries, resolve, now)...)
	}
	return &policy.ExceptionDecisions{Items: decisions}
}

func (s *localAssetScanner) checkResolver(local bool) exceptions.CheckResolver {
	if local {
		return checkResolver(s.compiledQueries)
	}
	// With an upstream, the asset's bundle is the upstream's: a check it does
	// not know cannot be excepted there.
	bundle, err := s.services.GetBundle(s.job.Ctx, &policy.Mrn{Mrn: s.job.Asset.Mrn})
	if err != nil {
		log.Warn().Err(err).Str("asset", s.job.Asset.Mrn).Msg("cannot fetch the asset's policies to look up excepted checks")
		return func(string) string { return "" }
	}
	return checkResolver(bundle.GetQueries())
}

// applyLocally puts every exception that is in effect on the asset policy, as
// the exception groups the resolver reads.
func (s *localAssetScanner) applyLocally(entries []*policy.ExceptionEntry, now time.Time) []*policy.ExceptionDecision {
	var apply []*policy.ExceptionEntry
	decisions := make([]*policy.ExceptionDecision, 0, len(entries))
	for _, e := range entries {
		d := &policy.ExceptionDecision{Entry: e}
		switch {
		case e.CheckMrn == "":
			d.Outcome = policy.ExceptionOutcome_EXCEPTION_OUTCOME_UNKNOWN_CHECK
			d.Reason = "no policy in this scan has this check"
		case isExpired(e, now):
			d.Outcome = policy.ExceptionOutcome_EXCEPTION_OUTCOME_EXPIRED
		default:
			d.Outcome = policy.ExceptionOutcome_EXCEPTION_OUTCOME_APPLIED
			apply = append(apply, e)
		}
		decisions = append(decisions, d)
	}

	if groups := exceptions.PolicyGroups(apply); len(groups) != 0 {
		if s.services.AssetExceptions == nil {
			s.services.AssetExceptions = map[string][]*policy.PolicyGroup{}
		}
		s.services.AssetExceptions[s.job.Asset.Mrn] = groups
	}
	return decisions
}

func isExpired(e *policy.ExceptionEntry, now time.Time) bool {
	return e.ValidUntil != nil && time.Unix(e.ValidUntil.Seconds, 0).Before(now)
}

// submittable splits entries into those that can be sent and the decisions
// for those that cannot: unknown checks, which the upstream would refuse, and
// expired ones, which are reported but do not apply anywhere.
func submittable(entries []*policy.ExceptionEntry, now time.Time) ([]*policy.ExceptionEntry, []*policy.ExceptionDecision) {
	var send []*policy.ExceptionEntry
	var decided []*policy.ExceptionDecision
	for _, e := range entries {
		switch {
		case e.CheckMrn == "":
			decided = append(decided, &policy.ExceptionDecision{
				Entry:   e,
				Outcome: policy.ExceptionOutcome_EXCEPTION_OUTCOME_UNKNOWN_CHECK,
				Reason:  "the asset's policies upstream have no such check",
			})
		case isExpired(e, now):
			decided = append(decided, &policy.ExceptionDecision{Entry: e, Outcome: policy.ExceptionOutcome_EXCEPTION_OUTCOME_EXPIRED})
		default:
			send = append(send, e)
		}
	}
	return send, decided
}

func notSubmitted(entries []*policy.ExceptionEntry, reason string) []*policy.ExceptionDecision {
	res := make([]*policy.ExceptionDecision, len(entries))
	for i, e := range entries {
		res[i] = &policy.ExceptionDecision{
			Entry:   e,
			Outcome: policy.ExceptionOutcome_EXCEPTION_OUTCOME_NOT_SUBMITTED,
			Reason:  reason,
		}
	}
	return res
}

const notAuthoritative = "not submitted: only a CI run on the default branch submits exceptions"

// declarations declares every source the matches came from with its complete
// key list, so that completeness is never inferred from one asset's subset.
func declarations(ms []matchedException) []*policy.ExceptionSourceDeclaration {
	seen := map[*policy.ExceptionSource]struct{}{}
	var res []*policy.ExceptionSourceDeclaration
	for _, m := range ms {
		src := m.match.Entry.Source
		if _, ok := seen[src]; ok {
			continue
		}
		seen[src] = struct{}{}
		res = append(res, &policy.ExceptionSourceDeclaration{
			Source:   src,
			Keys:     exceptions.AllKeys(m.source),
			Checksum: exceptions.SetChecksum(m.source),
		})
	}
	return res
}

// submitContext sends the context exceptions that matched this asset, with
// the full declaration of their source, before the asset is resolved.
func (s *localAssetScanner) submitContext(ms []matchedException, entries []*policy.ExceptionEntry, now time.Time) []*policy.ExceptionDecision {
	if len(entries) == 0 {
		return nil
	}
	send, decided := submittable(entries, now)
	if s.exceptions == nil || !s.exceptions.submit {
		return append(decided, notSubmitted(send, notAuthoritative)...)
	}
	return append(decided, s.submit(s.job.Asset.Mrn, send, declarations(ms))...)
}

// submitUser sends the user-scope exceptions once per run, against the space:
// they apply to every asset in it, so all of them are sent, not only those that
// govern the asset that happens to come first. Every asset reports the
// decisions of that one submission.
func (s *localAssetScanner) submitUser(entries []*policy.ExceptionEntry, resolve exceptions.CheckResolver, now time.Time) []*policy.ExceptionDecision {
	if len(entries) == 0 {
		return nil
	}
	send, decided := submittable(entries, now)
	if s.exceptions == nil || !s.exceptions.submit {
		return append(decided, notSubmitted(send, notAuthoritative)...)
	}

	cfg := s.exceptions
	cfg.userOnce.Do(func() {
		var all []*policy.ExceptionEntry
		for _, m := range exceptions.MatchAsset(cfg.user, "") {
			all = append(all, exceptions.ToProto(m, resolve))
		}
		sendAll, _ := submittable(all, now)
		decl := []*policy.ExceptionSourceDeclaration{{
			Source:   cfg.user[0].Source,
			Keys:     exceptions.AllKeys(cfg.user),
			Checksum: exceptions.SetChecksum(cfg.user),
		}}
		cfg.userDecisions = map[string]*policy.ExceptionDecision{}
		for _, d := range s.submit(s.job.UpstreamConfig.GetSpaceMrn(), sendAll, decl) {
			cfg.userDecisions[d.GetEntry().GetKey()] = d
		}
	})

	for _, e := range send {
		d, ok := cfg.userDecisions[e.Key]
		if !ok {
			d = &policy.ExceptionDecision{
				Entry:   e,
				Outcome: policy.ExceptionOutcome_EXCEPTION_OUTCOME_NOT_SUBMITTED,
				Reason:  "the upstream returned no decision for this exception",
			}
		}
		decided = append(decided, d)
	}
	return decided
}

// submit sends entries upstream and pairs every one with the upstream's
// decision. An upstream that does not take exceptions leaves none in effect.
func (s *localAssetScanner) submit(scopeMrn string, entries []*policy.ExceptionEntry, sources []*policy.ExceptionSourceDeclaration) []*policy.ExceptionDecision {
	if len(entries) == 0 && len(sources) == 0 {
		return nil
	}
	resp, err := s.services.SubmitExceptions(s.job.Ctx, &policy.SubmitExceptionsReq{
		ScopeMrn: scopeMrn,
		Entries:  entries,
		Sources:  sources,
	})
	if err != nil {
		reason := "not submitted: " + err.Error()
		if c := status.Code(err); c == codes.NotFound || c == codes.Unimplemented {
			reason = "not submitted: the upstream does not take exceptions"
		} else {
			log.Warn().Err(err).Str("scope", scopeMrn).Msg("failed to submit exceptions")
		}
		return notSubmitted(entries, reason)
	}
	return pairDecisions(entries, resp.GetDecisions())
}

// pairDecisions returns one decision per submitted entry, in submission order,
// matched by source and key. An entry the upstream said nothing about is not in
// effect.
func pairDecisions(entries []*policy.ExceptionEntry, got []*policy.ExceptionDecision) []*policy.ExceptionDecision {
	type id struct{ source, key string }
	byID := make(map[id]*policy.ExceptionDecision, len(got))
	for _, d := range got {
		byID[id{exceptions.SourceName(d.GetEntry().GetSource()), d.GetEntry().GetKey()}] = d
	}
	res := make([]*policy.ExceptionDecision, len(entries))
	for i, e := range entries {
		d, ok := byID[id{exceptions.SourceName(e.Source), e.Key}]
		if !ok {
			d = &policy.ExceptionDecision{
				Outcome: policy.ExceptionOutcome_EXCEPTION_OUTCOME_NOT_SUBMITTED,
				Reason:  "the upstream returned no decision for this exception",
			}
		}
		// report the entry as cnspec read it
		d = d.CloneVT()
		d.Entry = e
		res[i] = d
	}
	return res
}
