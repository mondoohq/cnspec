// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"testing"

	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// The Hosts & Devices forms that are not a login.
//
// local, ssh, winrm and filesystem are curated in
// cli/launcher/forms/forms_hosts.go and are exercised where the thing they are
// about lives: ssh in source_ambient_test.go for the known-hosts picker,
// filesystem and local in prefill_test.go. What is here is the pair that
// arrived in forms_misc_test.go -- a file whose list spanned five categories --
// and had nowhere else to be.

// hostsConnectors are the Hosts & Devices connectors this file covers.
var hostsConnectors = filedHere("device", "vagrant")

// vagrant is host-shaped but takes a machine name, not user@host, and carries
// no credential of its own.
func TestVagrantAsksForItsMachine(t *testing.T) {
	_, f := formFor(t, "vagrant")

	pos := positionalFields(&f)
	if len(pos) != 1 || !pos[0].Required {
		t.Fatalf("vagrant does not require a machine: %v", fieldLabels(f))
	}
	if pos[0].Label == "host" {
		t.Error(`the slot is still labelled "host", which reads as user@host`)
	}
	if !hasFlagField(f, "sudo") {
		t.Errorf("--sudo is declared but not offered: %v", fieldLabels(f))
	}

	pos[0].SetValue("default")
	if got := deliveryFor(f); got != deliverPlain {
		t.Errorf("vagrant declares no credential but delivery is %v", got)
	}
}

// device: the reason it is not curated, asserted rather than remembered.
//
// Every flag it declares is Hidden or Deprecated, so genericFields builds no
// field for any of them and applySpec's at() would return nil for every flag a
// spec could name -- silently, and while still satisfying the snapshot gate,
// which reads the recorded flag list including the hidden ones.
//
// If the os provider ever unhides them, this fails and device becomes worth a
// spec. That is the point of asserting it.
//
// That device is recorded as deliberately uncurated at all, rather than simply
// forgotten, is TestEveryConnectorIsCuratedOrExcused's business: it names the
// three exclusions over the whole catalog, so a fourth appearing silently is
// caught there rather than in whichever file its author had open.
func TestDeviceHasNoFlagLeftForASpecToName(t *testing.T) {
	c, f := formFor(t, "device")

	if len(c.Flags) == 0 {
		t.Fatal("device declares no flags at all; the snapshot is not what this reasoned about")
	}
	var visible []string
	for _, fl := range c.Flags {
		if fl.Option&plugin.FlagOption_Hidden == 0 && fl.Option&plugin.FlagOption_Deprecated == 0 {
			visible = append(visible, fl.Long)
		}
	}
	if len(visible) > 0 {
		t.Errorf("device now shows %v; curate it in cli/launcher/forms/forms_hosts.go instead of skipping it", visible)
	}
	if len(f.Fields()) > 0 {
		t.Errorf("device's form has %d fields, so there is something to curate: %v",
			len(f.Fields()), fieldLabels(f))
	}
}
