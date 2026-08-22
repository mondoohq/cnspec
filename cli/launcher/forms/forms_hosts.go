// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

// Hosts & Devices: device, filesystem, local, ssh, vagrant and winrm.
//
// The two shapes here are "this machine" and "a machine reached over the
// network", and both are curated the same way: name the target, put the
// credential in CREDENTIAL with its --ask-* partner first, and leave the rest
// as options. local is the degenerate case and declares an empty spec on
// purpose -- it takes no target at all, so the only thing its spec does is
// suppress the derived positional slot.
//
// ssh and winrm are the worked examples the network devices in
// forms_network.go were written from, and filesystem is the worked example for
// the path-shaped connectors elsewhere: "PATH" is the label a spec exists to
// replace, because the derived slot spells the usage string verbatim.
//
// device is deliberately left uncurated. The reason is in the init below rather
// than in this comment, because it is a claim about the connector that stops
// being true if the connector changes.

func init() {
	// The one connector this file owns and deliberately did not register a
	// spec for. It fails the same test artifactory does over in forms_dev.go: a
	// FormSpec is a set of claims about flags, and neither has flags a spec can
	// reach or that CI can check.
	//
	// device declares nine flags and every one of them is marked Hidden or
	// Deprecated, so genericFields skips all nine and the form has no field a
	// spec could name. applySpec's at() returns nil for a flag that was never
	// built, so a spec naming --lun or --device-name would be dropped in
	// silence -- while still passing TestEverySpecNamesRealFlags, which reads
	// the snapshot's flag list including the hidden ones. That is the
	// invisible-typo failure this phase exists to prevent, arrived at from the
	// other direction, so the honest answer is not to write the spec.
	// `cnspec scan device --help` shows the same emptiness: the CLI lists none
	// of them either, while the connector's own examples still say
	// `--lun <N>`.
	//
	// HasFormData() reports false for it now, which is the other half of the
	// same finding: it used to count the hidden flags, so the launcher drew an
	// empty form instead of the free-text argument box that is the honest
	// screen for a connector with nothing to ask.
	registerUncurated("device",
		"every flag it declares is Hidden or Deprecated, so no field exists for a spec to name")

	registerSpec("filesystem", FormSpec{
		Positional: []PositionalSpec{{
			Label: "path", Desc: "the mounted filesystem to scan",
			Required: true}},
	})

	// local takes no target at all -- it is this machine -- so it declares an
	// empty spec purely to suppress the derived positional slot. Promoting
	// anything into TARGET here would be misleading now that TARGET is the
	// first thing on the pane.
	registerSpec("local", FormSpec{})

	registerSpec("ssh", FormSpec{
		Positional: []PositionalSpec{{
			Label: "user@host", Desc: "the remote system to connect to",
			Required: true, Source: srcSSHHost}},
		Credential: []string{"ask-pass", "identity-file", "password"},
		Labels:     map[string]string{"ask-pass": "prompt for password"},
	})

	// vagrant is host-shaped but not a host connector: the argument is a
	// machine name out of the Vagrantfile, not user@host, and the connection
	// is made through `vagrant ssh-config` rather than by the launcher. It
	// declares no credential, so there is nothing for delivery.go to route.
	//
	// The machine names are enumerable -- `vagrant status --machine-readable`
	// lists them -- but a value picker is a Source, which lives in a file this
	// one does not own. Noted for whoever adds the enumerated sources.
	registerSpec("vagrant", FormSpec{
		Positional: []PositionalSpec{{
			Label: "machine", Desc: "the Vagrant machine name, as `vagrant status` lists it",
			Required: true,
		}},
		Labels: map[string]string{"sudo": "elevate privileges with sudo"},
	})

	registerSpec("winrm", FormSpec{
		Positional: []PositionalSpec{{
			Label: "user@host", Desc: "the remote Windows system to connect to",
			Required: true}},
		Credential: []string{"ask-pass", "password"},
		Labels:     map[string]string{"ask-pass": "prompt for password"},
	})
}
