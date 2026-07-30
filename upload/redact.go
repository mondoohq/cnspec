// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package upload

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
)

// redactedErrorMax bounds a reported error message. Mirrors the 512-byte cap
// already applied to upload response bodies before they are reported.
const redactedErrorMax = 512

// signedURLQueryRe matches the query string of any URL embedded in an error
// message. For a signed upload that query carries the signature — a
// time-limited write credential for the object — so it must never leave the
// host inside telemetry.
var signedURLQueryRe = regexp.MustCompile(`(https?://[^\s"?]*)\?[^\s"]*`)

// RedactError renders err for reporting to the platform with any URL removed.
//
// Transport errors from net/http arrive wrapped in *url.Error, which embeds the
// full request URL — and for a signed upload that URL's query string contains
// the signature granting write access to the object. Reporting it verbatim
// would copy a credential into error telemetry.
//
// Dropping the URL costs nothing diagnostically: the upload session id and
// asset MRN are already reported alongside as tags, and they identify the
// request far better than a single-use URL does.
//
// The result is truncated to redactedErrorMax bytes.
func RedactError(err error) string {
	if err == nil {
		return ""
	}

	var msg string
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// Keep the operation and the underlying cause, drop the URL entirely.
		msg = fmt.Sprintf("%s: %v", urlErr.Op, urlErr.Err)
	} else {
		msg = err.Error()
	}

	// Belt and braces: an error that is not a *url.Error can still have had a
	// URL formatted into it by an intermediate layer.
	msg = signedURLQueryRe.ReplaceAllString(msg, "$1?<redacted>")

	if len(msg) > redactedErrorMax {
		msg = msg[:redactedErrorMax] + "…"
	}
	return msg
}
