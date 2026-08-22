// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connectorgen

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/cockroachdb/errors"
)

// A regeneration must not lose a connector.
//
// The mql checkout is not a superset of what a person has installed. Three
// connectors in the recorded snapshot are not in the source tree at all:
// equinix ships as a built provider with no config package, and anthropic and
// clickhouse are earlier names for providers the tree now calls claude and
// clickhousedb. Overwriting the artifact with only what the source yields would
// delete three connectors' worth of metadata that was established by running
// the binaries, and the tests that depend on them would start skipping rather
// than failing -- which is the failure mode that costs the most to notice.
//
// So a regeneration keeps what it can no longer derive, marks it, and files a
// gap for it. A carried entry is visible in the diff and in the report every
// time, so it is a question that keeps getting asked rather than a fact that
// quietly rots.

// CarryForward adds connectors from a previous artifact that this extraction
// did not produce.
//
// It reads both the current envelope and the bare array the launcher's snapshot
// used before this tool existed, because the first regeneration reads the
// second.
func CarryForward(art *Artifact, previousPath string) error {
	data, err := os.ReadFile(previousPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrapf(err, "cannot read the previous artifact %s", previousPath)
	}

	previous, err := parsePrevious(data)
	if err != nil {
		return err
	}

	have := map[string]bool{}
	for _, c := range art.Connectors {
		have[c.Provider+"/"+c.Name] = true
	}

	var carried []Connector
	for _, c := range previous {
		if have[c.Provider+"/"+c.Name] {
			continue
		}
		c.CarriedForward = true
		carried = append(carried, c)
		art.Gaps = append(art.Gaps, Gap{
			Provider:  c.Provider,
			Connector: c.Name,
			Kind:      GapCarriedForward,
			Detail:    "not found in the source that was read; its metadata is the previously recorded one and nothing here re-derived it",
		})
	}
	if len(carried) == 0 {
		return nil
	}

	art.Connectors = append(art.Connectors, carried...)
	sort.SliceStable(art.Connectors, func(i, j int) bool {
		if art.Connectors[i].Name != art.Connectors[j].Name {
			return art.Connectors[i].Name < art.Connectors[j].Name
		}
		return art.Connectors[i].Provider < art.Connectors[j].Provider
	})
	sortGaps(art.Gaps)
	return nil
}

// parsePrevious reads either artifact format.
func parsePrevious(data []byte) ([]Connector, error) {
	var envelope Artifact
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Schema != 0 {
		return envelope.Connectors, nil
	}
	var bare []Connector
	if err := json.Unmarshal(data, &bare); err != nil {
		return nil, errors.Wrap(err, "the previous artifact is neither an artifact nor a connector list")
	}
	for i := range bare {
		if bare[i].Provider == "" || bare[i].Name == "" {
			return nil, fmt.Errorf("the previous artifact has an entry with no name or provider")
		}
	}
	return bare, nil
}
