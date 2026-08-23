// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package forms holds the curated connector overlays: what a connector's input
// screen asks for, in what order, and where a field's suggested values come
// from.
//
// Curated overlays. The generic layer in the launcher already produces a
// working screen for every connector from its declared metadata; an overlay
// improves the ones people reach for most, by naming the positional arguments
// the usage string only hints at, promoting the two or three fields that
// actually matter into TARGET, and attaching value pickers.
//
// An overlay never invents a flag. Every entry is matched against what the
// installed provider declares, and anything unmatched is dropped, so an overlay
// that goes stale degrades to the generic screen instead of emitting a flag
// that no longer exists. That graceful degradation is also why a typo is
// invisible at runtime, and why TestEverySpecNamesRealFlags exists.
//
// # One file per category, and a test that says so
//
// The specs are split across forms_<category>.go, one file per entry in
// cli/launcher/catalog's own taxonomy -- the same taxonomy the launcher groups
// the connector list by. A reader looking for the jamf form finds it where the
// UI shows jamf.
//
// The files used to be named forms_saas_a.go, forms_saas_b.go, forms_cloud_a.go
// and so on, where the letter was where one contributor's file ended and the
// next began. That kept two people from editing the same file and produced no
// structure at all: forms_cloud_a.go held alicloud through nutanix and
// forms_cloud_b.go held oci through vsphere, with the four virtualization
// connectors split across both for no reason a reader could recover.
//
// TestEverySpecIsFiledUnderItsCategory is what keeps the taxonomy a rule rather
// than a convention. registerSpec records the file each spec was registered
// from, and the test resolves each connector's category through
// catalog.Categorize -- so a spec added to the wrong file fails with the name of
// the file it belongs in.
//
// # Why this is its own package
//
// The specs need three things and none of them is a screen: the FormSpec shape,
// the source ids a picker is named by, and the launcher-owned field markers.
// The engine that turns a spec into fields (applySpec), the provider-derived
// half (providerFormSpec) and the merge between them stay in the launcher,
// because those read connector metadata and build widgets. What is here is the
// data and the registry it lives in.
//
// The import direction is forms -> {source, tui/form} and nothing more. source
// and tui/form know nothing about connectors or about this package, so a spec
// cannot reach a screen and a screen cannot reach back for a spec.
//
// # The six connectors with no snapshot entry
//
// auth0, bitwarden, dropbox, jumpcloud, keycloak and zoom reach the catalog only
// through the compiled-in static list.
//
// providers.DefaultProviders strips Flags, MinArgs, MaxArgs and Discovery, so
// on a machine without these providers installed a Connector for one carries
// nothing and HasFormData() is false: the pane says "open this connector to
// install its provider" and the spec is a no-op, because applySpec drops every
// flag the connector does not declare. The spec only starts mattering once the
// provider is installed -- and at that point the connector declares the flags
// named in it.
//
// They are absent from internal/connectors/connectors.json for the same reason,
// so TestEverySpecNamesRealFlags logs "no snapshot entry, so its flags were not
// checked" and moves on. That is a real gap and it is stated rather than papered
// over: those specs are checked by hand and by the launcher's own tests, not by
// the shared gate. The gap closes by itself the next time somebody refreshes the
// snapshot on a machine with these six installed:
//
//	cnspec providers install auth0 bitwarden dropbox jumpcloud keycloak zoom
//	go test ./apps/cnspec/cmd/interactive/ -run TestConnectorSnapshot -update
//
// What made curating them the better answer than leaving them generic is that
// the generic screen is actively wrong for four of them. The secret classifier
// puts --client-secret in CREDENTIAL and leaves --domain, --url, --account-id
// and --client-id in OPTIONS, which is collapsed behind a single row: the
// Auth0 tenant, the Keycloak server and the Zoom account -- the things being
// scanned -- are hidden, and the client id is separated from the secret it
// pairs with. And none of the six had a delivery route, so every one of them
// drew a credential field and then refused to launch.
//
// The classifier needs no correcting for any of them, which is why no Secret or
// NotSecret entry appears in those specs: --client-secret, --password, --token
// and --api-key are read as secrets, while --client-id, --account-id, --org-id
// and --username are read as the identifiers they are, and keycloak's --ca-cert
// is correctly public.
//
// # Credential delivery declares nothing here any more
//
// Four of the files this package was assembled from ended with a block of
// registered environment variables -- PORTAINER_ACCESS_TOKEN, IRU_TOKEN,
// ARTIFACTORY_TOKEN and ARTIFACTORY_API_KEY; JAMF_CLIENT_SECRET, the two
// MONGODB_ATLAS_*, NETLIFY_AUTH_TOKEN, NEXTDNS_API_KEY and
// OKTA_API_PRIVATE_KEY; TAILSCALE_OAUTH_CLIENT_SECRET, VERCEL_TOKEN,
// AUTH0_CLIENT_SECRET and seven more; ATLASSIAN_USER_TOKEN, DATABRICKS_TOKEN,
// DD_API_KEY and six more. Each was read out of the receiving provider's own
// ParseCLI and confirmed by running the provider with the credential missing,
// so that the provider named the variable itself. Each was still a claim that a
// particular provider reads a particular name, made by a person, checkable only
// by another person.
//
// None of them is needed now that the launcher calls ParseCLI rather than
// reading it, and the reasons those blocks gave are why the table could never
// have been right:
//
//   - artifactory's --token needs a `bearer` credential; a password credential
//     is spent as the legacy API key against the wrong header.
//   - jamf routes its credential on cred.User carrying the client id,
//     mongodbatlas on the labels "private-key" and "client-secret", and okta's
//     key wants a private_key credential rather than a password. An untagged
//     credential is dropped by all three without a word.
//   - tailscale reads the same credential as the API token or as the OAuth
//     client secret depending on whether --client-id is set, which is a decision
//     only tailscale can make and which a variable name cannot express.
//   - databricks resolves DATABRICKS_TOKEN itself, inside the SDK, when its own
//     credential switch falls through -- and that switch has no default arm, so
//     an untagged credential does not fail, it scans whatever account the
//     ambient variable names. The launcher refuses instead: it looks for the
//     value it sent in the asset that came back, and databricks either tagged it
//     or it is gone.
//   - helm's --password is the chart repository's, the provider reads no HELM_*
//     variable, and there is no --ask-pass to hand the prompt to the child, so a
//     typed credential with no registered route was refused outright.
//   - activedirectory reads LOGONSERVER, USERDNSDOMAIN, USERDOMAIN and USERNAME
//     to infer a controller and a principal, and reads nothing at all for the
//     password. There was no honest variable to register, so a typed password
//     was refused.
//
// All of them travel by inventory now, in whatever shape their own ParseCLI
// builds. Nothing in this package decides where a credential lands.
package forms
