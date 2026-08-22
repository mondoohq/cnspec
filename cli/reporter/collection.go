// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/json"
	"io"
	"os"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/policy"
)

// Every other output format in this package is a reduction: json-v1 and
// json-v2 keep scores and status, sarif and junit keep findings, the compact
// printers keep whatever fits on a terminal. None of them carry the bundle,
// the resolved policies, the query documentation or the raw llx results, so
// none of them can be read back into the structure a scan produced.
//
// json-full is that missing artifact. It is the whole *policy.ReportCollection
// and nothing else, so a consumer -- a report viewer, a diff, a re-render in
// another format -- gets exactly what the scanner had.
//
// Encoding is encoding/json rather than protojson, deliberately:
//
//   - It is the shape the rest of the repo already produces and consumes.
//     internal/scandump writes collections with json.MarshalIndent, and the
//     fixtures under testdata/ are read straight into policy.ReportCollection
//     with json.Unmarshal. Choosing protojson would fork the artifact away
//     from every collection dump that already exists.
//   - It round-trips losslessly for this message tree. The only oneof in
//     cnspec_policy.proto is on Metric, which ReportCollection does not
//     reference, and neither ReportCollection nor anything it reaches uses a
//     well-known type whose Go representation differs from its JSON one.
//     []byte fields (llx.Primitive.Value) go out as base64 and come back
//     byte-identical. TestCollectionRoundTrip proves this against both
//     fixtures with proto.Equal.
//   - protojson would additionally rename every field to lowerCamel, which
//     would make the artifact unreadable by the existing loaders above for no
//     gain in fidelity.
//
// Do not route this through (*ReportCollection).ToJSON(): it nils out
// Reports[k].Data on its own receiver, which is precisely the fidelity this
// format exists to keep.

// WriteCollection serializes the complete report collection to out.
func WriteCollection(data *policy.ReportCollection, out io.Writer) error {
	if data == nil {
		data = &policy.ReportCollection{}
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	// keep MQL snippets, URLs and markdown remediation readable instead of
	// escaping <, > and & into \u00xx sequences
	enc.SetEscapeHTML(false)

	if err := enc.Encode(data); err != nil {
		return errors.Wrap(err, "failed to write report collection")
	}
	return nil
}

// LoadCollection reads a json-full artifact back into a report collection.
//
// It also reads any other plain JSON serialization of a policy.ReportCollection,
// which includes the collections written by internal/scandump.
func LoadCollection(in io.Reader) (*policy.ReportCollection, error) {
	raw, err := io.ReadAll(in)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read report collection")
	}
	return UnmarshalCollection(raw)
}

// LoadCollectionFile reads a json-full artifact from a file path.
func LoadCollectionFile(path string) (*policy.ReportCollection, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read report collection from %s", path)
	}

	res, err := UnmarshalCollection(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load report collection from %s", path)
	}
	return res, nil
}

// UnmarshalCollection decodes a json-full artifact.
func UnmarshalCollection(raw []byte) (*policy.ReportCollection, error) {
	if err := rejectReducedReport(raw); err != nil {
		return nil, err
	}

	res := &policy.ReportCollection{}
	if err := json.Unmarshal(raw, res); err != nil {
		return nil, errors.Wrap(err, "failed to parse report collection")
	}
	return res, nil
}

// rejectReducedReport fails fast on the reduced json-v1 / json-v2 artifacts.
//
// Both of those are JSON objects whose keys partially overlap with
// ReportCollection's ("assets", "errors"), so decoding one would succeed and
// silently yield a collection with no reports rather than an error. A viewer
// cannot tell that apart from a scan where every asset failed, so the loader
// has to say so here.
func rejectReducedReport(raw []byte) error {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return errors.Wrap(err, "failed to parse report collection")
	}

	// neither key exists on ReportCollection; both exist on the reduced reports
	for _, key := range []string{"scores", "data"} {
		if _, ok := doc[key]; ok {
			return errors.New("this is a reduced report (json-v1/json-v2), not a full report collection; re-run the scan with --output json-full")
		}
	}
	return nil
}
