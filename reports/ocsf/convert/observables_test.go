// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// TestObservablesPerClass pins what a consumer can pivot on.
//
// The point of the attribute is that one query answers "everything about this
// host" across classes, so every class has to carry the asset's identity, not
// just the ones where it was convenient.
func TestObservablesPerClass(t *testing.T) {
	events := convertEvents(t, advisoryReportCollection(), Options{Findings: ocsf.FindingsCompliance})

	byClass := map[int][]ocsf.Observable{}
	for _, ev := range events {
		byClass[ev.class] = ev.observables(t)
	}
	require.Contains(t, byClass, ocsf.ClassUIDComplianceFinding)
	require.Contains(t, byClass, ocsf.ClassUIDVulnerabilityFinding)
	require.Contains(t, byClass, ocsf.ClassUIDInventoryInfo)

	for class, obs := range byClass {
		assert.Contains(t, obs, ocsf.Observable{
			Name: "device.uid", TypeID: ocsf.ObservableTypeResourceUID, Type: "Resource UID",
			Value: "//platformid.api.mondoo.app/hostname/X1",
		}, "class %d has to carry the asset identity", class)
		assert.Contains(t, obs, ocsf.Observable{
			Name: "device.hostname", TypeID: ocsf.ObservableTypeHostname, Type: "Hostname", Value: "X1",
		}, "class %d has to carry the hostname", class)
	}

	// The CVE is the pivot that makes "who else is exposed to this" one search.
	// Its path names the element it came from, which is what lets a consumer walk
	// back to the affected packages beside it.
	assert.Contains(t, byClass[ocsf.ClassUIDVulnerabilityFinding], ocsf.Observable{
		Name: "vulnerabilities[0].cve.uid", TypeID: ocsf.ObservableTypeCVEObjectUID,
		Type: "CVE Object: uid", Value: "CVE-2023-0286",
	})
}

// TestObservablesCarryTheAddress is what makes a finding joinable against
// netflow, firewall and EDR telemetry, which key on the address rather than on
// an asset id only Mondoo issues.
func TestObservablesCarryTheAddress(t *testing.T) {
	for _, ev := range convertEvents(t, sshAssetReportCollection(), Options{Findings: ocsf.FindingsCompliance}) {
		assert.Contains(t, ev.observables(t), ocsf.Observable{
			Name: "device.ip", TypeID: ocsf.ObservableTypeIPAddress, Type: "IP Address", Value: "10.0.0.4",
		}, "class %d", ev.class)
		assert.Contains(t, ev.observables(t), ocsf.Observable{
			Name:   "device.hostname",
			TypeID: ocsf.ObservableTypeHostname, Type: "Hostname", Value: "web-01.example.com",
		}, "class %d", ev.class)
	}
}

// TestObservablesDoNotLeakBetweenFindings covers the aliasing hazard in adding
// per-finding observables to the set shared by every event of an asset.
//
// The device observables are built once and handed to each event. Appending the
// CVEs to that slice rather than to a copy writes them into the backing array the
// next finding reads from, so findings accumulate each other's CVEs -- and the
// event still validates, because every path it names is real. Only the count
// gives it away.
func TestObservablesDoNotLeakBetweenFindings(t *testing.T) {
	events := convertEvents(t, advisoryReportCollection(), Options{Findings: ocsf.FindingsCompliance})

	for _, ev := range events {
		cves := 0
		for _, o := range ev.observables(t) {
			if o.TypeID == ocsf.ObservableTypeCVEObjectUID {
				cves++
			}
		}
		switch ev.class {
		case ocsf.ClassUIDVulnerabilityFinding:
			assert.Equal(t, 1, cves, "a finding for one advisory carries that advisory's CVE and no other")
		default:
			assert.Zero(t, cves, "class %d has no vulnerabilities attribute to point at", ev.class)
		}
	}
}

// TestObservablesOmitAbsentValues is the other half of the contract the OCSF
// validator enforces: an observable names an attribute that is in the event. An
// empty value is dropped by the encoder, so an observable for it would point at
// nothing.
func TestObservablesOmitAbsentValues(t *testing.T) {
	// A container image: no hostname to speak of, and an image reference where a
	// network connection would have an address.
	recorded, err := reportfixture.UbuntuScan()
	require.NoError(t, err)

	for _, ev := range convertEvents(t, recorded, Options{Findings: ocsf.FindingsDetection}) {
		for _, o := range ev.observables(t) {
			assert.NotEmpty(t, o.Value, "class %d carries an observable with no value", ev.class)
			assert.NotEqual(t, "device.hostname", o.Name, "a container image has no hostname")
			assert.NotEqual(t, "device.ip", o.Name, "an image reference is not an address")
		}
	}
}

func TestDeviceHostname(t *testing.T) {
	for _, tc := range []struct {
		name  string
		asset *inventory.Asset
		want  string
	}{
		{"fqdn wins", &inventory.Asset{
			Fqdn:        "web-01.example.com",
			PlatformIds: []string{hostnamePlatformID + "web-01"},
		}, "web-01.example.com"},
		{"platform id when there is no fqdn", &inventory.Asset{
			PlatformIds: []string{hostnamePlatformID + "web-01"},
		}, "web-01"},
		{"other platform ids are not hostnames", &inventory.Asset{
			PlatformIds: []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-abc"},
		}, ""},
		{"an empty hostname segment is not a hostname", &inventory.Asset{
			PlatformIds: []string{hostnamePlatformID},
		}, ""},
		{"nothing to go on", &inventory.Asset{}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, deviceHostname(tc.asset))
		})
	}
}

// TestDeviceIP pins that only an address becomes device.ip. A consumer matching
// findings against netflow or firewall logs joins on this field, and a hostname
// or an image reference filed under it matches nothing while looking like it
// should.
func TestDeviceIP(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hosts []string
		want  string
	}{
		{"ipv4", []string{"10.0.0.4"}, "10.0.0.4"},
		{"ipv6", []string{"2001:db8::1"}, "2001:db8::1"},
		{"a resolvable name is not an address", []string{"web-01.example.com"}, ""},
		{"an image reference is not an address", []string{"ubuntu:24.04"}, ""},
		{"a port is not part of the address", []string{"10.0.0.4:22"}, ""},
		{"the first address wins", []string{"web-01", "10.0.0.4", "10.0.0.5"}, "10.0.0.4"},
		{"no connections", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asset := &inventory.Asset{}
			for _, host := range tc.hosts {
				asset.Connections = append(asset.Connections, &inventory.Config{Host: host})
			}
			// A nil entry is reachable through the inventory and must not panic.
			asset.Connections = append(asset.Connections, nil)
			assert.Equal(t, tc.want, deviceIP(asset))
		})
	}
}

// convertEvents runs a scan through the converter and hands back each event with
// the class it belongs to, so a test can assert per class without re-deriving the
// mapping it is testing.
func convertEvents(t *testing.T, r *policy.ReportCollection, opts Options) []rawEvent {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, Convert(r, &buf, opts))

	var res []rawEvent
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		var ev struct {
			ClassUID    int               `json:"class_uid"`
			Observables []ocsf.Observable `json:"observables"`
		}
		require.NoError(t, json.Unmarshal(line, &ev))
		res = append(res, rawEvent{class: ev.ClassUID, obs: ev.Observables})
	}
	require.NotEmpty(t, res)
	return res
}

type rawEvent struct {
	class int
	obs   []ocsf.Observable
}

func (e rawEvent) observables(t *testing.T) []ocsf.Observable {
	t.Helper()
	assert.NotEmpty(t, e.obs, "class %d carries no observables", e.class)
	return e.obs
}
