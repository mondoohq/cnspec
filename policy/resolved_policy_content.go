// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"slices"
	"strconv"

	"google.golang.org/protobuf/proto"
)

// ContentChecksum fingerprints what a resolved policy makes an asset run and
// how its results are scored, independently of the bundle it was built from.
//
// GraphExecutionChecksum is the checksum of the whole bundle the asset's
// policy pulls in: every policy assigned to its space, whether or not the
// asset's filters admit it. buildResolvedPolicy derives every reporting-job
// UUID from it (rpBuilderData.relativeChecksum), and the collector job's own
// checksum hashes those UUIDs. So a change to a policy that never reaches an
// asset (a Linux-only query pack, for a macOS asset) gives that asset a
// resolved policy with a new graph checksum and new UUIDs everywhere, while
// it runs exactly the same queries and scores them exactly the same way.
//
// ContentChecksum is what does not depend on the bundle. It covers the
// execution job (every query by code ID: its checksum, properties and
// datapoints) and the collector job with every reporting-job UUID translated
// to the ID the job reports for: reporting jobs (type, scoring system,
// datapoints, child jobs with their impacts, notify targets, MRNs),
// datapoint types, reporting queries, risk MRNs, risk factors and risk data
// queries. Two resolved policies with the same ContentChecksum make the asset
// run the same queries and produce the same scores from them.
//
// The asset filters are not covered: they select the content, and the
// content is what is compared. Callers that care whether the filters moved
// compare FiltersChecksum themselves.
func (rp *ResolvedPolicy) ContentChecksum() string {
	if rp == nil {
		return ""
	}
	h := sha256.New()
	write := func(s string) {
		h.Write([]byte(strconv.Itoa(len(s))))
		h.Write([]byte{':'})
		h.Write([]byte(s))
	}
	write("resolved-policy-content-v1")

	// idOf translates a reporting-job UUID to the ID the job reports for. A
	// UUID no job carries keeps its raw value, so an inconsistent resolved
	// policy still hashes deterministically; it just cannot match one built
	// from a different bundle.
	idOf := map[string]string{}
	for uuid, rj := range rp.GetCollectorJob().GetReportingJobs() {
		idOf[uuid] = rj.GetQrId()
	}
	translate := func(uuid string) string {
		if id, ok := idOf[uuid]; ok {
			return "id:" + id
		}
		return "uuid:" + uuid
	}
	translated := func(uuids []string) []string {
		out := make([]string, 0, len(uuids))
		for _, u := range uuids {
			out = append(out, translate(u))
		}
		slices.Sort(out)
		return out
	}

	section := func(name string, entries []string) {
		slices.Sort(entries)
		write(name)
		write(strconv.Itoa(len(entries)))
		for _, e := range entries {
			write(e)
		}
	}

	var queries []string
	for codeID, q := range rp.GetExecutionJob().GetQueries() {
		e := contentEntry(codeID, q.GetChecksum())
		for _, k := range slices.Sorted(maps.Keys(q.GetProperties())) {
			e += contentEntry(k, q.GetProperties()[k])
		}
		datapoints := slices.Clone(q.GetDatapoints())
		slices.Sort(datapoints)
		e += contentEntry(datapoints...)
		queries = append(queries, e)
	}
	section("execution", queries)

	collector := rp.GetCollectorJob()

	var jobs []string
	for _, rj := range collector.GetReportingJobs() {
		e := contentEntry(rj.GetQrId(), rj.GetType().String(), rj.GetScoringSystem().String())
		e += contentEntry(slices.Sorted(maps.Keys(rj.GetDatapoints()))...)
		var children []string
		for childUUID, impact := range rj.GetChildJobs() {
			children = append(children, contentEntry(translate(childUUID), contentImpact(impact)))
		}
		slices.Sort(children)
		e += contentEntry(children...)
		e += contentEntry(translated(rj.GetNotify())...)
		mrns := slices.Clone(rj.GetMrns())
		slices.Sort(mrns)
		e += contentEntry(mrns...)
		jobs = append(jobs, e)
	}
	section("reporting_jobs", jobs)

	// A datapoint's notify list is left out: which jobs consume a datapoint is
	// already covered by each reporting job's own datapoints above, and the
	// notify list is not a stable copy of it. When two execution queries
	// produce the same datapoint, both list it, but notify names only the one
	// the builder saw first in map order, so two builds of the same content
	// can name different ones.
	var datapoints []string
	for checksum, info := range collector.GetDatapoints() {
		datapoints = append(datapoints, contentEntry(checksum, info.GetType()))
	}
	section("datapoints", datapoints)

	var reportingQueries []string
	for codeID, arr := range collector.GetReportingQueries() {
		reportingQueries = append(reportingQueries, contentEntry(codeID)+contentEntry(translated(arr.GetItems())...))
	}
	section("reporting_queries", reportingQueries)

	var riskMrns []string
	for uuid, arr := range collector.GetRiskMrns() {
		items := slices.Clone(arr.GetItems())
		slices.Sort(items)
		riskMrns = append(riskMrns, contentEntry(translate(uuid))+contentEntry(items...))
	}
	section("risk_mrns", riskMrns)

	var riskFactors []string
	for mrn, rf := range collector.GetRiskFactors() {
		b, err := proto.MarshalOptions{Deterministic: true}.Marshal(rf)
		if err != nil {
			// A risk factor that cannot be marshalled still has to change the
			// checksum, not collide with every other one that cannot.
			b = []byte("unmarshallable:" + err.Error())
		}
		riskFactors = append(riskFactors, contentEntry(mrn, string(b)))
	}
	section("risk_factors", riskFactors)

	var riskData []string
	for mrn, info := range collector.GetRiskDataQueries() {
		e := contentEntry(mrn)
		for _, k := range slices.Sorted(maps.Keys(info.GetDatapointChecksums())) {
			e += contentEntry(k, info.GetDatapointChecksums()[k])
		}
		riskData = append(riskData, e)
	}
	section("risk_data_queries", riskData)

	return hex.EncodeToString(h.Sum(nil))
}

// contentEntry frames its fields length-prefixed, so two different field
// lists never render the same string ("a"+"bc" vs "ab"+"c").
func contentEntry(fields ...string) string {
	out := strconv.Itoa(len(fields)) + "#"
	for _, f := range fields {
		out += strconv.Itoa(len(f)) + ":" + f
	}
	return out
}

func contentImpact(impact *Impact) string {
	if impact == nil {
		return "nil"
	}
	value := "nil"
	if impact.GetValue() != nil {
		value = strconv.Itoa(int(impact.GetValue().GetValue()))
	}
	return contentEntry(value, impact.GetScoring().String(), strconv.Itoa(int(impact.GetWeight())), impact.GetAction().String())
}
