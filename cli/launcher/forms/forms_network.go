// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

// Network & Security Devices: arista, bigip, checkpoint, ciscocatalyst,
// fortios, host, ipinfo, ipmi, junos, mikrotik, nd-ssh, networkdiscovery, nmap,
// opcua, panos, redfish, shodan and unifi.
//
// The category name is misleading -- these do not share a shape -- and the
// three shapes it actually covers are all here.
//
// Twelve are appliances: a box on the network, a user, a password. Five take
// the device as a positional (`arista user@host`, `ipmi USER@HOST`,
// `mikrotik user@host`, `ciscocatalyst hostname`, `host HOST`); the rest
// address it through a --hostname flag with --port and --username beside it
// (bigip, checkpoint, fortios, junos), or through their own spelling of the
// same thing (nd-ssh, panos, redfish, opcua, unifi). All of them are curated
// the way ssh and winrm are in forms_hosts.go: name the target, put the
// credential in CREDENTIAL with its --ask-* partner first, and leave the rest
// as options.
//
// nmap and networkdiscovery take a network or a domain as the *input* to a scan
// and have no credential at all. shodan and ipinfo are API services reached
// with a token and no host.
//
// Every spec below was written from that connector's own declaration in
// ~/.config/mondoo/providers/<name>/<name>.json and its provider source, not
// from the category it is filed under, and cross-checked against
// internal/connectors/connectors.json, which is what TestEverySpecNamesRealFlags
// reads. Note that ciscocatalyst and nd-ssh are served by the `networkdevices`
// provider and host by `network`, so the provider file is not always named
// after the connector.
//
// Two things decided most of what is here.
//
// The first is that --networks is not a discovery picker. Both nmap and shodan
// declare it, and it reads like somewhere a list of discovered ranges would
// go. It is the opposite: it is what the scan is pointed *at*. Attaching a
// picker to it would offer the answer as the question.
//
// The second is that a credential the receiving provider does not keep makes a
// form that refuses at the moment the user presses enter. Every credential
// named below was checked against the provider that receives it, which is now
// something the launcher does for itself at launch rather than something
// recorded here -- see TestEveryCredentialFieldRoundTrips.
//
// # Exactly one ssh-config picker in this file, and it is nd-ssh's
//
// srcSSHHost lists the Host aliases in ~/.ssh/config, and the temptation is to
// hang it off every `user@host` field here. It is attached to nd-ssh and to
// nothing else, because for the others it would be offering values that do not
// work:
//
//   - arista connects with goeapi over HTTPS -- verified in the provider's own
//     connection.go, which calls goeapi.Connect("https", host, user, secret,
//     port). It is not an SSH client at all.
//   - ipmi talks RMCP to a BMC on port 623 (verified likewise: the connection
//     defaults conf.Port to 623). A BMC has its own address, separate from the
//     SSH address of the OS running on the same chassis.
//   - mikrotik uses the RouterOS API service on 8728/8729, and bigip,
//     checkpoint and fortios are REST APIs. None of them is reached by ssh.
//   - ciscocatalyst addresses a Catalyst Center appliance through its API. Its
//     --password description mentions SSH, but that flag set is shared with the
//     sibling nd-ssh connector in the same networkdevices provider, and nd-ssh
//     is where the SSH client in that binary belongs -- which is why nd-ssh has
//     the picker and ciscocatalyst does not.
//   - host is not a login at all; see its spec below.
//
// junos is the one genuine remaining candidate -- NETCONF over SSH on port 830,
// and --identity-file really is an SSH private key. It is still left off,
// because sshHosts() returns the alias from the `Host` line rather than the
// resolved HostName, and an alias only means something to a client that reads
// ~/.ssh/config. cnspec's os provider does; there is no evidence the junos
// provider does, and a picker whose values get dialled literally as DNS names
// would fail more often than it helped. If someone confirms junos resolves
// ssh_config aliases, attaching srcSSHHost to its --hostname is a one-line
// change here.

func init() {
	// arista takes the device as its single argument and declares nothing else
	// but the password pair, so there is no TARGET beyond the positional.
	registerSpec("arista", FormSpec{
		Positional: []PositionalSpec{{
			Label: "user@host", Desc: "the Arista EOS device to connect to",
			Required: true,
		}},
		Credential: []string{"ask-pass", "password"},
		Labels:     map[string]string{"ask-pass": "prompt for password"},
	})

	// bigip is addressed by flag rather than by argument: `bigip` takes no
	// positional at all, and the device is --hostname.
	registerSpec("bigip", FormSpec{
		Target:     []string{"hostname", "port", "username"},
		Credential: []string{"ask-pass", "password"},
		Labels: map[string]string{
			"port":     "iControl REST port",
			"username": "user",
			"ask-pass": "prompt for password",
			"insecure": "skip TLS verification",
		},
	})

	// checkpoint scans a management server, not a gateway -- the gateways are
	// what --discover finds behind it, and the generic layer already builds
	// that field from the connector's declared discovery targets.
	//
	// --api-key is deliberately absent from Credential. It is the documented
	// alternative to username/password, but the connector declares no
	// --ask-api-key to prompt for it and no environment variable that the
	// provider is confirmed to read, so there is no verified route to deliver
	// it and this file does not claim one. The classifier still marks it a
	// secret and still puts it in CREDENTIAL; filling it produces the
	// launcher's honest refusal rather than a credential sent somewhere that
	// may ignore it.
	registerSpec("checkpoint", FormSpec{
		Target:     []string{"hostname", "port", "domain", "username"},
		Credential: []string{"ask-pass", "password"},
		Labels: map[string]string{
			"hostname":    "management server",
			"port":        "Management API port",
			"domain":      "management domain",
			"username":    "user",
			"ask-pass":    "prompt for password",
			"insecure":    "skip TLS verification",
			"fingerprint": "pin TLS fingerprint (SHA-1)",
		},
	})

	// ciscocatalyst is `ciscocatalyst hostname`: the Catalyst Center appliance
	// is the argument, and --user is the account on it. --discover devices is
	// how the switches it manages are reached.
	registerSpec("ciscocatalyst", FormSpec{
		Positional: []PositionalSpec{{
			Label: "hostname", Desc: "the Cisco Catalyst Center to connect to",
			Required: true,
		}},
		Target:     []string{"user"},
		Credential: []string{"ask-pass", "password"},
		Labels: map[string]string{
			"user":     "user",
			"ask-pass": "prompt for password",
		},
	})

	// fortios has no Credential section, and that is the whole story of this
	// entry. Its only credential is --token, which the classifier puts in
	// CREDENTIAL by itself, so a Credential list would only be repeating what
	// already happened.
	//
	// It used to be an unrouted credential, and the reasoning is worth keeping
	// because it says something true about the provider: fortios declares no
	// --ask-token, and while FORTIOS_ACCESS_TOKEN appears in the shipped binary
	// it sits in a block with FORTIOS_ACCESS_USERNAME, FORTIOS_ACCESS_PASSWORD
	// and FORTIOS_CA_* -- variables for flags this connector does not declare
	// -- which places that block in the vendored FortiOS SDK rather than in the
	// provider's own ParseCLI. None of that decides anything now: the token is
	// handed to that ParseCLI as --token, and the round-trip test confirms
	// fortios keeps it as a bearer credential.
	//
	// --enable-forti-sdk-logs is a debugging switch and is hidden.
	registerSpec("fortios", FormSpec{
		Target: []string{"hostname"},
		Hide:   []string{"enable-forti-sdk-logs"},
		Labels: map[string]string{
			"token":    "REST API token",
			"insecure": "skip TLS verification",
		},
	})

	// host is the odd one in this family: it is a scanning target, not a
	// login. The network provider's ParseCLI splits the argument into scheme,
	// host, port and path and connects to whatever is listening, so this asks
	// for a URL or a host and nothing else. It declares no credential flag at
	// all, which is why there is no CREDENTIAL section to give it.
	registerSpec("host", FormSpec{
		Positional: []PositionalSpec{{
			Label: "host", Desc: "hostname, URL or host:port to inspect",
			Required: true,
		}},
		Labels: map[string]string{
			"insecure":         "skip TLS verification",
			"follow-redirects": "follow HTTP redirects",
		},
	})

	// unifi declares no --ask-api-key and reads no environment variable of its
	// own, and its --api-key travels regardless.
	//
	// Two workarounds preceded that. The first was a registered variable, which
	// unifi has none of. The second was MONDOO_API_KEY, the name mql derives
	// from any flag with no config mapping -- verified against the shipped
	// binary at the time: with nothing set, `cnspec scan unifi --hostname
	// 127.0.0.99` failed in ParseCLI with "username is required (or use
	// --api-key)", and with the variable set it got past ParseCLI and failed on
	// the TLS connection instead. Both were routes into req.Flags. The launcher
	// fills req.Flags directly now, so neither is needed and neither can be
	// shadowed by a configuration key that happens to share a name.

	// Not curated, and deliberately: ipinfo.
	//
	// The connector declares no flags, no positional arguments and no
	// discovery targets -- `cnspec scan ipinfo` is the whole command line, and
	// an IP address is passed to the resource in MQL rather than to the
	// connector. HasFormData is therefore false for it, which is also why it
	// is absent from internal/connectors/connectors.json. A FormSpec would have nothing
	// to name: applySpec drops every flag the connector does not declare, so
	// each entry would be silently ignored.
	//
	// Its one credential, IPINFO_TOKEN, has no flag either -- the connection
	// reads the variable directly and nothing on the command line carries it --
	// so there is nothing for a form to collect and nothing for ParseCLI to be
	// handed. It is left on the generic form, where the launcher inherits the
	// user's own environment and the variable works as documented.
	registerUncurated("ipinfo",
		"it declares no flags, no arguments and no discovery targets, so a spec would have nothing to name")

	// ipmi addresses the BMC, which is a different machine from the OS on the
	// same hardware and has its own address. `ipmi USER@HOST` carries the
	// account in the argument, so --password is all that is left.
	registerSpec("ipmi", FormSpec{
		Positional: []PositionalSpec{{
			Label: "user@host", Desc: "the BMC or service processor to connect to",
			Required: true,
		}},
		Credential: []string{"ask-pass", "password"},
		Labels:     map[string]string{"ask-pass": "prompt for password"},
	})

	// junos is ssh's closest relative here -- NETCONF over SSH on port 830 --
	// so its CREDENTIAL is ordered the way ssh's is: the prompt first, then
	// the key, then the password. --identity-file is a path rather than a
	// secret, and the classifier already reads it that way, so naming it in
	// Credential moves it into the section without making it a masked field.
	registerSpec("junos", FormSpec{
		Target:     []string{"hostname", "port", "username"},
		Credential: []string{"ask-pass", "identity-file", "password"},
		Labels: map[string]string{
			"port":          "NETCONF port",
			"username":      "user",
			"ask-pass":      "prompt for password",
			"identity-file": "SSH private key",
		},
	})

	// mikrotik declares MinArgs=1 with the usage string
	// `mikrotik user@host [flags]`, which the derived layer cannot line up: two
	// tokens against one argument falls through to the single-argument case and
	// labels the box "user@host [flags]". Naming the positional here is what
	// fixes that.
	//
	// The two ports are the RouterOS API service and its TLS twin, and which
	// one applies depends on --tls. They are offered as suggestions, not as a
	// validation list -- Choices on a flag leaves the field free text, the same
	// way the aws region list does.
	registerSpec("mikrotik", FormSpec{
		Positional: []PositionalSpec{{
			Label: "user@host", Desc: "the MikroTik RouterOS device to connect to",
			Required: true,
		}},
		Target:     []string{"port"},
		Credential: []string{"ask-pass", "password"},
		Choices:    map[string][]string{"port": {"8728", "8729"}},
		Labels: map[string]string{
			"port":     "RouterOS API port",
			"ask-pass": "prompt for password",
			"tls":      "use the API-SSL service",
			"insecure": "skip TLS verification",
		},
	})

	// nd-ssh is `ssh` for network gear: the same transport, the same
	// user@host, and the same ~/.ssh/config to suggest from. The picker is a
	// suggestion rather than a list of valid values, so a device that is not
	// in that file is still typed in as normal.
	//
	// Three of its flags carry a secret, and the two prompts are promoted over
	// the two boxes that would collect the same values by hand.
	//
	// The route no longer forces that choice -- one ParseCLI call carries as
	// many secrets as the form holds -- but the screen is still the better one.
	// --ask-pass is built into the CLI and --ask-enable-password carries
	// FlagOption_AskInput, so the child prompts for each and sets the flag
	// itself; the value never exists outside the process that uses it, which is
	// stronger than anything the keychain can offer. A Cisco login password and
	// enable password are also two boxes nobody wants to fill twice.
	registerSpec("nd-ssh", FormSpec{
		Positional: []PositionalSpec{{
			Label: "user@host", Desc: "the network device to connect to",
			Required: true, Source: srcSSHHost,
		}},
		Credential: []string{"ask-pass", "password", "ask-enable-password", "private-key-path"},
		Labels: map[string]string{
			"ask-pass":              "prompt for password",
			"ask-enable-password":   "prompt for enable password",
			"private-key-path":      "private key",
			"system-transport-args": "extra ssh transport options",
		},
		// --enable-password and --private-key-passphrase are covered by the
		// prompts above; --store-commands writes every command run on the
		// device to a file for debugging, which is not a launcher's business.
		Hide: []string{"enable-password", "private-key-passphrase", "store-commands"},
	})

	// networkdiscovery declares no flags at all: one domain in, subdomains
	// out. The only thing worth curating is that the enumeration is the point
	// of the connector rather than an option, so --discover leads the form
	// instead of sitting behind the fold.
	registerSpec("networkdiscovery", FormSpec{
		Positional: []PositionalSpec{{
			Label: "domain", Desc: "the FQDN whose subdomains to enumerate",
			Required: true,
		}},
		Target: []string{"discover"},
		Labels: map[string]string{"discover": "what to enumerate"},
	})

	// nmap and shodan answer the same three questions -- one host, one domain,
	// a range of addresses -- through the same `host <TARGET>` / `domain
	// <TARGET>` sub-commands plus a --networks flag, and neither shape is
	// visible in the generic form: MaxArgs=2 with no argument names in the
	// usage string renders one box labelled "argument 1".
	//
	// The leading selector is what makes the three shapes separate screens.
	// "network range" emits nothing, because a range is passed as --networks
	// rather than as a sub-command word, and the flag is shown only for it:
	// the provider ignores --networks once a sub-command names a single
	// target.
	registerSpec("nmap", FormSpec{
		Positional: []PositionalSpec{
			{
				Label: "what to scan", Desc: "one host, a domain, or a range of addresses",
				Required: true,
				Options:  []string{"host", "domain", "network range"},
				Emit: map[string]string{
					"host": "host", "domain": "domain", "network range": "",
				},
			},
			{
				Label: "target", Desc: "IP address or hostname",
				Required: true, ShowIf: []string{"host", "domain"},
			},
		},
		Target: []string{"networks", "ports"},
		Labels: map[string]string{
			"networks": "address ranges",
			"ports":    "ports to scan",
		},
		// --networks is the input to the scan, not a place to put something
		// discovery found, so it gets no picker.
		ShowFlagsIf: map[string][]string{"networks": {"network range"}},
	})

	// opcua is a single field. It matters only that --endpoint carries
	// FlagOption_Required: a required flag left where the classifier put it
	// sits in OPTIONS, behind the "more" fold, where the form asks for
	// something the user cannot see.
	registerSpec("opcua", FormSpec{
		Target: []string{"endpoint"},
		Labels: map[string]string{"endpoint": "endpoint URL"},
	})

	// panos and redfish are the simple case: exactly one secret flag, and an
	// --ask-pass partner for it, so the child prompts and nothing is carried.
	// panos addresses the device by --hostname and redfish by a user@host
	// argument, which is the only real difference between their forms.
	registerSpec("panos", FormSpec{
		Target:     []string{"hostname", "username"},
		Credential: []string{"ask-pass", "password"},
		Labels: map[string]string{
			"hostname": "device",
			"username": "user",
			"ask-pass": "prompt for password",
			"insecure": "skip TLS verification",
		},
	})

	registerSpec("redfish", FormSpec{
		Positional: []PositionalSpec{{
			Label: "user@host", Desc: "the management controller to connect to",
			Required: true,
		}},
		Credential: []string{"ask-pass", "password"},
		Labels: map[string]string{
			"ask-pass": "prompt for password",
			"insecure": "skip TLS verification",
		},
	})

	// shodan is the same sub-command shape with an account behind it: with no
	// argument at all the target is the Shodan account itself, which is what
	// the connector's own summary calls it, so that is a fourth option rather
	// than an empty form.
	//
	// --token has no --ask-token partner, so it travels in SHODAN_TOKEN, which
	// the provider reads in both its ParseCLI and its connection. That entry
	// is already registered in delivery.go; nothing is added for it here.
	registerSpec("shodan", FormSpec{
		Positional: []PositionalSpec{
			{
				Label: "what to query", Desc: "the account, one host, a domain, or a range of addresses",
				Required: true,
				Options:  []string{"account", "host", "domain", "network range"},
				Emit: map[string]string{
					"account": "", "host": "host", "domain": "domain", "network range": "",
				},
			},
			{
				Label: "target", Desc: "IP address or hostname",
				Required: true, ShowIf: []string{"host", "domain"},
			},
		},
		Target:      []string{"networks"},
		Credential:  []string{"token"},
		Labels:      map[string]string{"token": "API token", "networks": "address ranges"},
		ShowFlagsIf: map[string][]string{"networks": {"network range"}},
	})

	// unifi is the one connector here with two ways in, and the connector's
	// own help says they are alternatives: an API key, or a username and
	// password. Showing both at once is not just noise -- deliveryFor carries
	// one secret, so a form holding both has nowhere to send either. The
	// selector keeps them on separate screens.
	registerSpec("unifi", FormSpec{
		Positional: []PositionalSpec{{
			Label: "sign in with", Desc: "how to authenticate to the controller",
			Required: true,
			Options:  []string{"username and password", "API key"},
			Emit:     map[string]string{"username and password": "", "API key": ""},
		}},
		Target:     []string{"hostname", "port", "site", "username"},
		Credential: []string{"ask-pass", "password", "api-key"},
		Labels: map[string]string{
			"hostname": "controller",
			"username": "user",
			"site":     "site",
			"ask-pass": "prompt for password",
			"api-key":  "API key",
			"insecure": "skip TLS verification",
		},
		ShowFlagsIf: map[string][]string{
			"username": {"username and password"},
			"password": {"username and password"},
			"ask-pass": {"username and password"},
			"api-key":  {"API key"},
		},
	})
}
