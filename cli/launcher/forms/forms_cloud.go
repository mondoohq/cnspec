// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

// The Cloud & Virtualization family: the three big clouds people reach for
// first -- aws, azure and gcp -- the smaller ones addressed by a token or a
// named local profile, and the four hypervisors.
//
// They divide into three shapes, and the specs below are built around the
// division rather than around who wrote them:
//
//   - Account-wide clouds (aws, azure, gcp, alicloud, digitalocean, equinix,
//     hcp, hetzner, oci, stackit). The target is an account, a project or a
//     tenancy, and the useful screen leads with the thing that selects one --
//     a profile, a subscription, a project id -- rather than with a host.
//   - OpenStack, which is addressed by a project and authenticated with a
//     token or a password, and is neither a public cloud nor a host.
//   - The hypervisors (nutanix, proxmox, vcd, vsphere). Each is addressed by a
//     host, and its credential is a password or an API token for that host.
//     They were split across two files before this one and are together now,
//     because "a box you log in to" is what they have in common and the letter
//     in the old file name was not.
//
// Every flag named below was read out of internal/connectors/connectors.json,
// which is the recorded copy of what the installed providers declare. Nothing
// here is inferred from a usage string or from what a sibling cloud happens to
// spell its flags -- applySpec drops a flag the connector does not declare, so
// an invented name produces a field that silently never appears.
//
// The credential notes below were established by running the provider, not by
// reading its prose, and several of them are kept even though the route they
// describe no longer exists: they record what a provider does with a value it
// is handed, which is still what decides whether a launch is safe. What has
// changed is that the launcher asks the provider rather than consulting a table
// written from those transcripts. Where no route exists, the flag is left out
// of Credential and said so, rather than pointed at a plausible-looking
// variable the provider never reads.

// awsRegions is a display list, not a validation list: the region flag stays
// free text so a region newer than this binary still works.
var awsRegions = []string{
	"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
	"ap-south-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
	"ap-northeast-2", "ca-central-1", "sa-east-1",
}

// alicloudRegions is a display list, not a validation list: the region flag
// stays free text so a region newer than this binary still works. Every id here
// was found in the installed alicloud provider's own string table.
var alicloudRegions = []string{
	"cn-hangzhou", "cn-shanghai", "cn-beijing", "cn-shenzhen", "cn-qingdao",
	"cn-zhangjiakou", "cn-huhehaote", "cn-wulanchabu", "cn-chengdu",
	"cn-heyuan", "cn-guangzhou", "cn-hongkong",
	"ap-southeast-1", "ap-southeast-3", "ap-southeast-5", "ap-southeast-6",
	"ap-southeast-7", "ap-northeast-1", "ap-northeast-2", "ap-south-1",
	"us-east-1", "us-west-1", "eu-central-1", "eu-west-1",
	"me-east-1", "me-central-1",
}

// ociRegions is a display list, not a validation list: the region flag stays
// free text so a region newer than this binary still works. The entries are the
// commercial-realm (oc1) identifiers from the OCI Go SDK's own regions.json,
// trimmed to the ones people actually reach for -- the full realm has 46.
var ociRegions = []string{
	"us-ashburn-1", "us-phoenix-1", "us-chicago-1", "us-sanjose-1",
	"ca-toronto-1", "ca-montreal-1",
	"eu-frankfurt-1", "eu-amsterdam-1", "eu-zurich-1", "eu-madrid-1",
	"eu-paris-1", "eu-milan-1", "eu-stockholm-1",
	"uk-london-1", "uk-cardiff-1",
	"ap-tokyo-1", "ap-osaka-1", "ap-seoul-1", "ap-singapore-1",
	"ap-sydney-1", "ap-melbourne-1", "ap-mumbai-1",
	"sa-saopaulo-1", "sa-vinhedo-1", "sa-santiago-1",
	"me-dubai-1", "me-jeddah-1", "il-jerusalem-1", "af-johannesburg-1",
}

// ociAuthMethods mirrors connection.SupportedAuthMethods in the oci provider,
// which ParseCLI validates --auth-method against and rejects anything outside
// of. A value not on this list is a hard error before the scan starts, so this
// is one of the few lists that really is the whole set.
var ociAuthMethods = []string{
	"api-key", "instance-principal", "resource-principal",
	"workload-identity", "security-token",
}

func init() {
	// alicloud authenticates with an AccessKey pair. --access-key-id is not a
	// secret -- the classifier reads the "-id" suffix as a reference, and it is
	// half of a pair whose other half is the credential -- so it travels on the
	// command line while the secret travels in the environment. That split was
	// checked end to end rather than assumed:
	//
	//	ALIBABA_CLOUD_ACCESS_KEY_SECRET=… cnspec shell alicloud \
	//	    --region cn-hangzhou --access-key-id …
	//	→ SDKError … Code: InvalidAccessKeyId.NotFound
	//
	// which is the STS round trip refusing a bogus key, i.e. the pair arrived.
	// Dropping the flag from that same command instead gives
	//
	//	unable to get credentials … Access key ID must be specified via
	//	environment variable (ALIBABA_CLOUD_ACCESS_KEY_ID)
	//
	// so the provider really is combining the flag with the variable.
	//
	// --sts-token is the one credential in this file the provider throws away,
	// and the launcher now says so rather than guessing around it. alicloud's
	// own ParseCLI keeps --access-key-id, --region, --regions, --role-arn and
	// --role-session-name and drops --sts-token entirely -- confirmed by
	// sending it and looking for it in the asset that came back, which is what
	// TestEveryCredentialFieldRoundTrips does and records. A launch that fills
	// only that box is refused with a message naming the flag.
	//
	// It was previously left unrouted on weaker evidence: ALIBABA_CLOUD_SECURITY_TOKEN
	// is in the binary's string table, and probing it produced the same
	// InvalidAccessKeyId.NotFound with and without the variable, which proved
	// nothing either way.
	//
	// A hand-written inventory did not work either, and that was worth checking
	// rather than assuming: a file carrying options.access-key-id and
	// options.access-key-secret scans and then fails with the same "Access key
	// ID must be specified via environment variable" -- the provider never
	// reads those options back. Which option key it *does* read is alicloud's
	// business, and asking its ParseCLI is how the launcher now finds out.
	//
	// The profile leads TARGET, and it is the one field here that is not a
	// flag: alicloud declares no --profile, so the chosen profile travels to
	// the child as ALIBABA_CLOUD_PROFILE. See the note at the bottom of this
	// file for why that needed a seam rather than a spelling.
	registerSpec("alicloud", FormSpec{
		Positional: []PositionalSpec{{
			Label:   "profile",
			Desc:    "a profile from ~/.alibabacloud/credentials",
			Special: SpecialAlicloudProfile,
			Source:  srcAlicloudProfile,
		}},
		Target:     []string{"region", "regions", "role-arn"},
		Credential: []string{"access-key-id", "access-key-secret"},
		Choices:    map[string][]string{"region": alicloudRegions},
		Labels: map[string]string{
			"access-key-id":     "AccessKey ID",
			"access-key-secret": "AccessKey secret",
			"sts-token":         "STS security token",
			"role-arn":          "assume RAM role",
			"role-session-name": "RAM role session name",
			"region":            "region",
			"regions":           "regions to scan (optional)",
		},
	})

	registerSpec("aws", FormSpec{
		Target:  []string{"profile", "region", "role"},
		Sources: map[string]string{"profile": srcAWSProfile},
		Choices: map[string][]string{"region": awsRegions},
		Labels: map[string]string{
			"profile": "profile", "region": "region", "role": "assume role",
		},
	})

	registerSpec("azure", FormSpec{
		Target:     []string{"subscriptions"},
		Credential: []string{"auth-method", "tenant-id", "client-id", "client-secret", "certificate-path", "certificate-secret", "federated-token-file"},
		Choices: map[string][]string{
			"auth-method": {"cli", "env", "workload-identity", "managed-identity"},
		},
		Labels: map[string]string{"subscriptions": "subscriptions"},
	})

	// digitalocean is a token and nothing else: the account is the target, and
	// what varies is which of its resources to look at. So --discover is the
	// TARGET here rather than an option behind the "more" fold, which is the
	// rule this file follows for a connector that declares no scoping flag at
	// all. Promoting it anywhere a real target flag exists would only push that
	// flag down.
	//
	// The credential widgets -- the readout naming DIGITALOCEAN_TOKEN, the
	// paste box over --token, and the second, report-only readout for the
	// Spaces keys -- are declared in source_ambient.go and are not repeated
	// here. DIGITALOCEAN_TOKEN is registered there too, so --token already has
	// its route.
	registerSpec("digitalocean", FormSpec{
		Target:     []string{"discover"},
		Credential: []string{"token"},
		// --token is deliberately not relabelled. applyAmbient redraws it as
		// the paste box and gives it the guarantee as its description, and the
		// flag's own name is what the readout above it is about; a second name
		// for the same thing only makes the pair harder to read.
		Labels: map[string]string{"filters": "filter discovered assets"},
	})

	// equinix takes a sub-command pair -- `equinix org <id>` or `equinix
	// project <id>` -- which MinArgs=2 records and the usage string only hints
	// at. The two words come from the connector's own help and are emitted
	// verbatim.
	//
	// PACKET_AUTH_TOKEN is already in the delivery registry, and the provider
	// names it itself: without it the command refuses, and with it
	//
	//	PACKET_AUTH_TOKEN=… cnspec shell equinix project …
	//	→ GET https://api.equinix.com/metal/v1/projects: 401 Invalid authentication token
	//
	// which is the API rejecting a bogus token, i.e. the token arrived.
	registerSpec("equinix", FormSpec{
		Positional: []PositionalSpec{
			{
				Label: "kind", Desc: "an organization, or a single project",
				Required: true,
				Options:  []string{"org", "project"},
			},
			{
				Label: "id", Desc: "the organization or project id",
				Required: true,
			},
		},
		Credential: []string{"token"},
		Labels:     map[string]string{"token": "API token"},
	})

	// gcp follows the same shape as k8s: ask what is being scanned, then only
	// what that answer needs. The sub-command words come from the connector's
	// own help -- `gcp project <ID>`, `gcp org <ID>`, `gcp instance <NAME>
	// --project-id <ID> --zone <ZONE>` -- and are emitted verbatim.
	registerSpec("gcp", FormSpec{
		Positional: []PositionalSpec{
			{
				Label: "what to scan", Desc: "the kind of Google Cloud resource",
				Required: true,
				Options: []string{
					"project", "organization", "folder",
					"compute instance", "compute snapshot", "container registry",
				},
				Emit: map[string]string{
					"project":            "project",
					"organization":       "org",
					"folder":             "folder",
					"compute instance":   "instance",
					"compute snapshot":   "snapshot",
					"container registry": "gcr",
				},
			},
			{
				Label: "id", Desc: "project, organization, folder id or resource name",
				Required: true,
				SourceBy: map[string]string{
					"project":            srcGCPProject,
					"container registry": srcGCPProject,
				},
				// gcloud config names one project; the full list is a live
				// call, fetched when the picker is opened. It belongs only to
				// the kinds that take a project -- an organization id is not
				// one of them.
				LiveSourceBy: map[string]string{
					"project":            srcGCPProjectAll,
					"container registry": srcGCPProjectAll,
				},
			},
		},
		Target:      []string{"project-id", "zone", "repository"},
		Sources:     map[string]string{"project-id": srcGCPProject, "zone": srcGCPZone},
		LiveSources: map[string]string{"project-id": srcGCPProjectAll},
		Labels: map[string]string{
			"project-id":       "project",
			"credentials-path": "service account key",
		},
		// A single instance or snapshot is addressed by project and zone; a
		// registry by its repository. Nothing else needs either.
		ShowFlagsIf: map[string][]string{
			"project-id": {"compute instance", "compute snapshot"},
			"zone":       {"compute instance", "compute snapshot"},
			"repository": {"container registry"},
		},
		Credential: []string{"credentials-path"},
	})

	// hcp connects with a service principal. The client id is not a secret and
	// travels as a flag; the client secret travels in HCP_CLIENT_SECRET, which
	// was confirmed as the exact mixture the launcher produces:
	//
	//	cnspec shell hcp --client-id …
	//	→ HCP credentials required: set --client-id and --client-secret
	//	HCP_CLIENT_SECRET=… cnspec shell hcp --client-id …
	//	→ failed to get new token: oauth2: "unauthorized" "Authentication failed."
	//
	// The second reached HashiCorp's token endpoint, so the flag and the
	// variable were combined into one credential.
	//
	// --org-id and --project-id scope the connection and are optional: with
	// neither, the connection is the service principal's whole organization.
	// Neither gets a picker -- discover.hcp.projects is a declared id with no
	// registered source, excluded in source_discover_test.go because a
	// discovered project's id is composed at runtime and nothing shows where it
	// lands. Naming it here would fail TestEverySourceNamedByASpecExists.
	registerSpec("hcp", FormSpec{
		Target:     []string{"org-id", "project-id"},
		Credential: []string{"client-id", "client-secret"},
		Labels: map[string]string{
			"org-id":        "organization (optional)",
			"project-id":    "project (optional)",
			"client-id":     "service principal client id",
			"client-secret": "service principal client secret",
		},
	})

	// hetzner is the same shape as digitalocean: one token, and the only
	// question left is what to enumerate, so --discover leads. --endpoint
	// stays an option -- it points the client at a different Hetzner Cloud API,
	// which almost nobody wants and which would otherwise sit above the thing
	// the form is for.
	//
	// The readout and paste box come from source_ambient.go, and HCLOUD_TOKEN
	// is registered there.
	registerSpec("hetzner", FormSpec{
		Target:     []string{"discover"},
		Credential: []string{"token"},
		// As digitalocean: the paste box names itself.
		Labels: map[string]string{"endpoint": "API endpoint (optional)"},
	})

	// nutanix reaches a Prism Central instance, so the endpoint is the target
	// and it is the connector's one required flag -- named in TARGET so it is
	// reachable without opening the "more" fold, where focusFirstMissing would
	// otherwise park a cursor that cannot be moved to it.
	//
	// It authenticates two ways -- basic auth, or an IAM API key -- and neither
	// has an environment variable to travel in. NUTANIX_API_KEY,
	// NUTANIX_PASSWORD and PRISM_CENTRAL_API_KEY were all tried against
	// `cnspec shell nutanix --endpoint …` and all three left the provider
	// saying "missing credentials: provide --user with --password/--ask-pass,
	// or --api-key".
	//
	// Neither needs one. Both travel by inventory, in whatever shape the
	// provider's own ParseCLI builds for them, and --ask-pass remains a toggle
	// the user can tick to be prompted instead.
	registerSpec("nutanix", FormSpec{
		Target:     []string{"endpoint", "port", "user"},
		Credential: []string{"ask-pass", "password", "api-key"},
		Labels: map[string]string{
			"endpoint": "Prism Central host",
			"user":     "username",
			"ask-pass": "prompt for password",
			"api-key":  "IAM API key",
			"insecure": "skip TLS verification",
		},
	})

	// oci has three ways in and the form has to show all of them without
	// implying they combine: a named profile out of ~/.oci/config, an explicit
	// API key (tenancy + user + region + fingerprint + key file), or one of the
	// keyless principal flows selected with --auth-method.
	//
	// The scope is --tenancy. --filters narrows what is *inside* a tenancy --
	// its keys are regions=, compartments=, tag:, and the exclude: forms of
	// each, checked against filterKeyPrefixes in the provider -- so it is not a
	// second way to name an account and gets no discovery picker.
	//
	// --key-secret is the passphrase for an encrypted API key, and it is named
	// here now that it has somewhere to go. It was left out while three things
	// were true and still are: oci declares no --ask-key-secret, reads no
	// environment variable of its own anywhere in its provider or connection
	// code, and its inventory shape cannot be produced from this form --
	// NewOciConnection requires conf.Credentials[0] to be a private_key
	// credential carrying both the key and the passphrase, while --key-path is
	// classified as a plain reference and never becomes a credential at all.
	//
	// None of that is a problem the launcher has to solve any more. The
	// connector's own ParseCLI builds whatever credential oci wants out of the
	// flags it is handed, including the private_key shape this form could never
	// have produced, and the launcher writes back what it gets.
	registerSpec("oci", FormSpec{
		Target:     []string{"profile", "region", "tenancy", "filters"},
		Credential: []string{"auth-method", "config-file", "user", "fingerprint", "key-path", "key-secret"},
		Sources:    map[string]string{"profile": srcOCIProfile},
		Choices:    map[string][]string{"region": ociRegions, "auth-method": ociAuthMethods},
		Labels: map[string]string{
			"profile":     "config profile",
			"config-file": "config file",
			"tenancy":     "tenancy OCID",
			"user":        "user OCID",
			"fingerprint": "API key fingerprint",
			"key-path":    "API key file",
			"key-secret":  "API key passphrase",
			"auth-method": "authentication method",
			"filters":     "filters (regions, compartments, tags)",
		},
	})

	// openstack is a Keystone endpoint plus a scope. Either half can come from
	// a clouds.yaml entry named by --cloud, which is why --cloud leads TARGET:
	// choosing one answers the auth URL, the project and the region at once.
	//
	// Both of its secrets travel in the environment. resolveAuth in the
	// provider hands everything to gophercloud's clientconfig, whose v2auth and
	// v3auth read OS_PASSWORD and OS_APPLICATION_CREDENTIAL_SECRET whenever the
	// corresponding field is still empty -- which is exactly the state the
	// launcher leaves them in when it strips the secret off the command line.
	// The inventory route is *not* available here and must not be claimed: the
	// provider comments say passwords are deliberately never read back out of
	// conf.Options, and conn.Options is where a generated inventory would put
	// one, since openstack spells its user flag --username and the launcher
	// only recognises --user.
	registerSpec("openstack", FormSpec{
		Target: []string{
			"cloud", "auth-url", "region",
			"project-name", "project-id", "project-domain-name", "project-domain-id",
		},
		Credential: []string{
			"username", "password", "user-domain-name", "user-domain-id",
			"application-credential-id", "application-credential-name",
			"application-credential-secret",
		},
		Labels: map[string]string{
			"cloud":                         "clouds.yaml entry",
			"auth-url":                      "Keystone auth URL",
			"project-name":                  "project",
			"project-id":                    "project ID",
			"project-domain-name":           "project domain",
			"project-domain-id":             "project domain ID",
			"user-domain-name":              "user domain",
			"user-domain-id":                "user domain ID",
			"application-credential-id":     "application credential ID",
			"application-credential-name":   "application credential",
			"application-credential-secret": "application credential secret",
			"insecure":                      "skip TLS verification",
		},
	})

	// proxmox is a host URL and an API token. It declares no --ask-token and
	// reads no environment variable of its own -- there is no os.Getenv
	// anywhere in the provider, and no PROXMOX_* string in the shipped binary
	// -- so an inventory file is the only way its token travels, which is what
	// every credential now does.
	registerSpec("proxmox", FormSpec{
		Target:     []string{"host"},
		Credential: []string{"token"},
		Labels: map[string]string{
			"host":     "host URL",
			"token":    "API token",
			"insecure": "skip TLS verification",
		},
	})

	// stackit is one project, named by --project-id. That is required *input*,
	// not something to discover: NewStackitConnection refuses to build without
	// it, and there is no API to list projects before you have authenticated to
	// one. So no picker is attached to it.
	//
	// --service-account-key is a JSON key blob and the shared classifier does
	// not see it: the name ends in "-key", which is neither a strong secret
	// word nor a reference, so without the override below the whole key would
	// go on the command line where `ps auxww` reads it. The two "-path"
	// variants are genuinely paths and are correctly left unmarked.
	registerSpec("stackit", FormSpec{
		Target: []string{"project-id", "region", "endpoint"},
		Credential: []string{
			"token", "service-account-key", "service-account-key-path",
			"private-key", "private-key-path",
		},
		Secret: []string{"service-account-key"},
		Labels: map[string]string{
			"project-id":               "project",
			"token":                    "service account token",
			"service-account-key":      "service account key (JSON)",
			"service-account-key-path": "service account key file",
			"private-key":              "service account private key",
			"private-key-path":         "service account private key file",
		},
	})

	// vcd names its host and user as flags rather than as a positional, and
	// marks both FlagOption_Required -- so they have to be promoted into TARGET
	// here or they sit in OPTIONS behind the fold, where a cursor parked on a
	// missing required field cannot be reached.
	//
	// --ask-pass stays an ordinary toggle on the form. Ticking it makes the
	// child prompt, which is the best route there is because the value never
	// exists outside the process that uses it; typing the password instead
	// carries it by keychain. What the launcher no longer does is substitute
	// the first for the second.
	registerSpec("vcd", FormSpec{
		Target:     []string{"host", "user", "organization"},
		Credential: []string{"ask-pass", "password"},
		Labels: map[string]string{
			"ask-pass":     "prompt for password",
			"organization": "organization (optional)",
		},
	})

	// vsphere is addressed by a single positional argument, and its shape is
	// the reason this spec exists at all: ParseCLI prepends "scheme://" to the
	// argument and hands it to url.Parse, so the user half is everything before
	// the *last* @ and vCenter's realm form -- chris@vsphere.local@host --
	// parses as user "chris@vsphere.local" on host "host". There is no flag for
	// it and there is no separate realm flag, so a spec that mapped the value
	// onto one would be inventing something the connector does not declare. It
	// is a positional field, labelled to say what belongs in it, and an
	// optional :port rides along the same way url.Parse takes one.
	//
	// Like vcd, the password rides --ask-pass and needs no registration.
	registerSpec("vsphere", FormSpec{
		Positional: []PositionalSpec{{
			Label:    "user@realm@host",
			Desc:     "vCenter or ESXi host, with the user and realm to log in as",
			Required: true,
		}},
		Credential: []string{"ask-pass", "password"},
		Labels:     map[string]string{"ask-pass": "prompt for password"},
	})
}

// How srcAlicloudProfile is attached, and why it took a change to the contract.
//
// The source reads ~/.alibabacloud/credentials and declares
// Env: ALIBABA_CLOUD_PROFILE, so a chosen profile can travel to a connector
// that has no --profile flag. Attaching it needs a field, and until
// PositionalSpec.Special existed the only field shape a spec could add for a
// flagless value was a positional -- which was exactly the problem:
//
//   - args() emits every visible positional whose emitted() is non-empty, so
//     the profile would also be appended to the command line. alicloud declares
//     MinArgs=0 and MaxArgs=0, and `cnspec shell alicloud staging` answers
//     `unknown command "staging" for "cnspec shell alicloud"`. The form would
//     assemble a command that cannot run.
//   - the obvious escape, an empty Emit map, suppressed the argument by making
//     emitted() return "" for every value -- but formEnvironment builds the
//     variable out of emitted() too, so the child would get
//     ALIBABA_CLOUD_PROFILE= instead. An empty variable is not an absent one;
//     the SDK reads it as an explicit empty profile.
//
// Both halves of the seam read the same accessor, so no spelling of a spec
// carried the value without also emitting it. Special is the missing half: it
// marks the field as the launcher's, which args() already skips for both
// positionals and flags, while leaving emitted() -- and therefore
// formEnvironment -- untouched. The same seam is what attaches DOCKER_CONTEXT
// to docker and container in forms_container.go.
//
// Leaving the field empty is still the old behaviour and still correct: no
// variable is set, and whatever the credentials file and ALIBABA_CLOUD_PROFILE
// already say in the user's own shell is inherited by the child untouched.
