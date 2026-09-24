// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// The observables every event carries.
//
// An observable is a pivot: it names an attribute elsewhere in the event and
// repeats its value in one flat, typed list, so a SIEM can ask "everything about
// this host" or "every asset with this CVE" without knowing that a hostname lives
// at device.hostname on one class and somewhere else on another. OCSF marks the
// attribute recommended on every class cnspec emits.
//
// Two rules hold for every observable built here, because the OCSF validator
// enforces both: name has to be a path that exists on the class, and value has to
// be what the event actually carries at that path. That is why these are derived
// from the built event rather than from the asset a second time -- a value that
// drifted from the attribute it points at would be worse than no observable.

package convert

import (
	"slices"
	"strconv"

	"go.mondoo.com/cnspec/reports/ocsf"
)

// deviceObservables are the pivots every class shares: the asset's identity and
// the two ways a SIEM knows an endpoint.
//
// device is on all four classes, so one builder serves all of them. The asset MRN
// is also on resources[].uid for the finding classes, but pointing at the copy
// that every class has keeps one observable rather than two carrying the same
// value under different paths.
func deviceObservables(device *ocsf.Device) []ocsf.Observable {
	if device == nil {
		return nil
	}
	var res []ocsf.Observable
	for _, o := range []struct {
		name   string
		typeID int
		value  string
	}{
		{"device.uid", ocsf.ObservableTypeResourceUID, device.UID},
		{"device.hostname", ocsf.ObservableTypeHostname, device.Hostname},
		{"device.ip", ocsf.ObservableTypeIPAddress, device.IP},
	} {
		// An empty value is omitted from the event by the encoder, so an
		// observable naming it would point at an attribute that is not there.
		if o.value == "" {
			continue
		}
		res = append(res, observable(o.name, o.typeID, o.value))
	}
	return res
}

// withCVEObservables adds one observable per CVE in the finding, which is what
// makes "who else is exposed to this CVE" a single search rather than a scan of
// every vulnerabilities array in the index.
//
// The base slice is shared by every event of the asset, so it is cloned rather
// than appended to: append would write the first finding's CVEs into the backing
// array the next finding reads from.
func withCVEObservables(base []ocsf.Observable, vulns []ocsf.Vulnerability) []ocsf.Observable {
	res := slices.Clone(base)
	for i, vuln := range vulns {
		if vuln.CVE == nil || vuln.CVE.UID == "" {
			continue
		}
		// The index is the one that resolves: the path has to point at the
		// element this observable came from.
		name := "vulnerabilities[" + strconv.Itoa(i) + "].cve.uid"
		res = append(res, observable(name, ocsf.ObservableTypeCVEObjectUID, vuln.CVE.UID))
	}
	return res
}

// observable fills in the caption from the identifier, so the pair cannot
// disagree.
func observable(name string, typeID int, value string) ocsf.Observable {
	return ocsf.Observable{
		Name:   name,
		TypeID: typeID,
		Type:   ocsf.ObservableTypeName(typeID),
		Value:  value,
	}
}
