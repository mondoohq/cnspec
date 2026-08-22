// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"go.mondoo.com/cnspec/cli/tui"
	"go.mondoo.com/mql/cli/config"
)

// Where a scan's results go is not a detail. A scan that reports to Mondoo
// Platform sends what it found off this machine; an incognito one keeps it
// here. `cnspec scan` decides that silently -- it warns into the log when there
// are no credentials and switches to incognito -- and a launcher that hides the
// log has to say it on screen instead.

// upstreamState is what the launcher knows about reporting, and it is
// deliberately only what can be known cheaply.
type upstreamState struct {
	// configured is true when this machine has credentials to report with.
	configured bool
	// scope names the space those credentials belong to, when the config says.
	scope string
	// incognito is the user's choice, and is forced when configured is false.
	incognito bool

	// modal is the chooser that makes the choice, because the choice and the
	// only thing that can change it are the same subject. See upstreammodal.go.
	modal upstreamModalState
}

// reporting reports whether a scan launched now would send its results
// upstream.
func (u upstreamState) reporting() bool { return u.configured && !u.incognito }

// canToggle reports whether the choice is the user's to make. Without
// credentials there is nothing to toggle to.
func (u upstreamState) canToggle() bool { return u.configured }

// readUpstreamFn is the config read, replaceable in tests: the interesting
// states are "no credentials" and "credentials for a named space", and a test
// must not depend on which of those the developer happens to have.
var readUpstreamFn = readUpstream

// readUpstream asks the same question `cnspec scan` asks, in the one way that
// does not touch the network.
//
// scan.go calls opts.GetServiceCredential(), which for the ssh and wif
// authentication methods performs a *token exchange* -- ExchangeSSHKey and
// ExchangeExternalToken both dial the API. Calling that to draw a badge would
// put a network round trip on the render path and freeze the UI on a bad link,
// which is the same defect the keychain write had. So this reads the config and
// asks whether credentials are declared, which is a file read and nothing more.
//
// The consequence is worth stating: a declared credential can still turn out to
// be invalid. This says "configured to report", not "proven to report", and the
// scan itself remains the thing that finds out.
func readUpstream() upstreamState {
	opts, err := config.Read()
	if err != nil || opts == nil {
		return upstreamState{incognito: true}
	}
	configured := opts.ServiceAccountMrn != "" ||
		opts.PrivateKey != "" ||
		opts.Certificate != "" ||
		opts.Authentication != nil

	return upstreamState{
		configured: configured,
		scope:      spaceName(opts.GetScopeMrn()),
		incognito:  !configured,
	}
}

// badge says where a scan's results would go. It leads the header's right side
// because it is the one thing on screen that decides whether what the scan finds
// leaves this machine.
//
// Reporting is green and filled; incognito is hollow and dim. When there are no
// credentials the badge is dimmer still and carries no key hint, because there
// is nothing to toggle to -- pressing it explains rather than doing nothing.
func (u upstreamState) badge() string {
	if u.reporting() {
		label := "connected"
		if u.scope != "" {
			label += " " + u.scope
		}
		return tui.StyleGood.Render("● " + label)
	}
	if u.canToggle() {
		return tui.StyleWarn.Render("◌ incognito")
	}
	return tui.StyleFaint.Render("◌ incognito")
}

// badgeWidth is what the header reserves for the badge, so the clickable zone
// and the drawn glyphs agree.
func (u upstreamState) badgeWidth() int { return tui.Width(u.badge()) }

// spaceName is the last segment of a space MRN, which is the part a person
// recognises. An MRN is long and mostly boilerplate:
// //captain.api.mondoo.app/spaces/keen-tesla-123456 -> keen-tesla-123456
func spaceName(mrn string) string {
	if mrn == "" {
		return ""
	}
	for i := len(mrn) - 1; i >= 0; i-- {
		if mrn[i] == '/' {
			return mrn[i+1:]
		}
	}
	return mrn
}
