// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package form

import (
	"strings"

	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// Identifying which flags carry a secret is the single most important thing
// the form layer does, because a value the launcher gets wrong here ends up in
// the process table where every user on the machine can read it.
//
// FlagOption_Password cannot be the answer. A sweep of the installed provider
// set found 60 secret-carrying flags across 45 providers that leave Option
// unset -- azure --client-secret, github --token, okta --private-key,
// alicloud --access-key-secret, and so on. Only the database family and os/ssh
// set the bit. So the bit is one signal among several, and the classifier errs
// toward marking: a false positive costs a temp file, a false negative leaks a
// credential.
//
// The word lists below are shared by every connector, which makes them the
// wrong place to fix one connector's odd flag: widening a word to catch it
// re-classifies every other flag that happens to contain it. A spec says so
// about its own flag instead, with FormSpec.Secret and FormSpec.NotSecret, and
// those override everything here.

// strongSecretWords end a flag name whose value is unambiguously a credential.
// A name ending this way is a secret even when it also mentions a certificate:
// --certificate-secret is the passphrase for the certificate, not the
// certificate.
var strongSecretWords = []string{
	"secret", "password", "passwd", "passphrase", "token", "api-key", "apikey",
}

// weakSecretWords suggest a credential but are just as often part of the name
// of something that merely points at one, so they are only trusted after the
// reference checks below have had their say.
var weakSecretWords = []string{"private-key", "credential"}

// referenceWords mark a flag that names *where* a credential lives rather than
// holding one: a path, a file, an identifier, a mode. Those are safe on the
// command line and have to stay there, because that is how the provider
// expects to receive them.
var referenceWords = []string{
	"-path", "-file", "-name", "-id", "-mode", "-method", "-url", "-type",
}

// IsSecretFlag reports whether a flag's value must be kept off the command
// line.
//
// It takes no connector, and that is the design rather than an omission: every
// per-connector correction is a FormSpec.Secret or FormSpec.NotSecret entry in
// the file that curates the connector, beside the reason it is needed --
// datadog's --app-key, stackit's --service-account-key, okta's --private-key.
// A name here would invite those corrections back into word lists that every
// other connector's flags are read through.
func IsSecretFlag(fl plugin.Flag) bool {
	// A toggle or a number cannot carry a secret. This is what keeps the
	// --ask-pass and --ask-api-key family -- whose whole purpose is to make the
	// child process prompt, the safest option available -- from being misread.
	if fl.Type == plugin.FlagType_Bool || fl.Type == plugin.FlagType_Int {
		return false
	}
	if fl.Option&plugin.FlagOption_Password != 0 {
		return true
	}
	if IsSecretReference(fl) {
		return false
	}

	name := strings.ToLower(fl.Long)
	for _, w := range strongSecretWords {
		if strings.HasSuffix(name, w) || strings.Contains(name, w+"-") {
			return true
		}
	}
	// Certificate and CA material is public by construction, and only reaches
	// here once the strong words above have been ruled out.
	if strings.Contains(name, "cert") || strings.HasSuffix(name, "-ca") {
		return false
	}
	for _, w := range weakSecretWords {
		if strings.Contains(name, w) {
			return true
		}
	}
	return false
}

// IsSecretReference reports whether a flag names a file rather than holding a
// value. The name usually says so, but not always -- github's
// --app-private-key is a path and only its description admits it -- so the
// description is consulted too.
func IsSecretReference(fl plugin.Flag) bool {
	name := strings.ToLower(fl.Long)
	for _, ref := range referenceWords {
		if strings.Contains(name, ref) {
			return true
		}
	}
	desc := strings.ToLower(fl.Desc)
	if strings.Contains(desc, "path to") || strings.Contains(desc, "file path") ||
		strings.Contains(desc, "path of") {
		return true
	}
	return false
}
