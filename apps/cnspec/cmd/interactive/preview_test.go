// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
)

// The command bar says "$ <command>", and a user reads it to decide whether to
// press the button. It was allowed to be wrong in two ways at once, and both
// were the same mistake: a second place that assembled the words.
//
// plan() puts --incognito on the command through verb() when the user has asked
// to keep the results on this machine. The preview built its own prefix out of
// []string{"cnspec", a.Name, c.Name} and could not know, so the bar showed a
// command that reported upstream while the button ran one that did not. The
// inventory route went further and dropped the flag from the *command*: it
// assembled []string{a.Name, "--inventory-file", path} directly, so a user who
// chose incognito on the one route that carries a credential got a scan that
// reported upstream anyway.
//
// The cases below are one per route, run with the choice both ways, because a
// route that never carries the flag and a route that always does are both
// wrong in a way a single-value test would miss. There are two routes now
// rather than four, and the one that matters is still the one carrying the
// credential.
func TestPreviewAndLaunchAgreeOnTheIncognitoChoice(t *testing.T) {
	// The keychain is asked for the inventory route, and a locked one blocks on
	// an OS dialog. Failing it takes the documented fallback -- the 0600 file --
	// which is the branch that assembles the argument list either way.
	orig := storeCredentialFn
	storeCredentialFn = func(id string, cred *vault.Credential) error {
		return errors.New("keyring unavailable")
	}
	defer func() { storeCredentialFn = orig }()

	for _, tc := range []struct {
		name   string
		conn   Connector
		fill   func(t *testing.T, f *form)
		secret string
		want   delivery
	}{
		{
			name: "plain",
			conn: awsConnector(),
			fill: func(t *testing.T, f *form) { fieldByLabel(t, *f, "profile").SetValue("prod") },
			want: deliverPlain,
		},
		{
			name: "inventory, host connector",
			conn: sshConnector(),
			fill: func(t *testing.T, f *form) {
				fieldByLabel(t, *f, "user@host").SetValue("chris@10.0.0.4")
				fieldByLabel(t, *f, "password").SetValue(sentinel)
			},
			secret: "password",
			want:   deliverInventory,
		},
		{
			name:   "inventory, API token",
			conn:   githubConnector(),
			fill:   func(t *testing.T, f *form) { fieldByLabel(t, *f, "personal access token").SetValue(sentinel) },
			secret: "token",
			want:   deliverInventory,
		},
	} {
		for _, incognito := range []bool{false, true} {
			f := newForm(tc.conn)
			tc.fill(t, &f)
			r := launchRequest{form: f, incognito: incognito}
			withParser(t, &fakeParser{secretFlag: tc.secret})

			if got := deliveryFor(f); got != tc.want {
				t.Fatalf("%s: route = %v, want %v -- the case no longer covers what it names", tc.name, got, tc.want)
			}

			preview := r.preview(tc.conn, scanAction())
			plan, err := r.plan(tc.conn, scanAction())
			if plan.cleanup != nil {
				defer plan.cleanup()
			}
			if err != nil {
				t.Fatalf("%s (incognito=%v): plan: %v", tc.name, incognito, err)
			}

			inPreview := strings.Contains(preview, "--incognito")
			inCommand := containsString(plan.args, "--incognito")
			if inPreview != inCommand {
				t.Errorf("%s (incognito=%v): the bar shows --incognito=%v and the command has it=%v\n  bar: %s\n  cmd: cnspec %s",
					tc.name, incognito, inPreview, inCommand, preview, strings.Join(plan.args, " "))
			}
			if inCommand != incognito {
				t.Errorf("%s: the user chose incognito=%v and the command reads %v",
					tc.name, incognito, strings.Join(plan.args, " "))
			}
			if strings.Contains(preview, sentinel) {
				t.Errorf("%s: the bar showed the secret: %q", tc.name, preview)
			}
		}
	}
}

// The route whose words are fully known before anything is written has to match
// the command exactly, not merely agree about one flag.
//
// The inventory route is deliberately excluded: its path is created by plan(),
// so the bar names "<generated, 0600>" instead. That difference is the intended
// one -- a path shown before the file exists would be a lie by the time it was
// read -- and everything ahead of it is checked above.
func TestPreviewIsTheCommandWordForWord(t *testing.T) {
	conn := awsConnector()
	for _, incognito := range []bool{false, true} {
		f := newForm(conn)
		fieldByLabel(t, f, "profile").SetValue("prod")
		r := launchRequest{form: f, incognito: incognito}

		plan, err := r.plan(conn, scanAction())
		if plan.cleanup != nil {
			defer plan.cleanup()
		}
		if err != nil {
			t.Fatalf("plan: %v", err)
		}
		want := "cnspec " + strings.Join(plan.args, " ")
		if got := r.preview(conn, scanAction()); got != want {
			t.Errorf("incognito=%v:\n bar %q\n cmd %q", incognito, got, want)
		}
	}
}

// The bar names the file rather than a path, because writing the file is what
// pressing the button does.
//
// It also no longer says "cnspec cannot carry --x safely yet" for anything.
// That message was the preview's way of showing a refusal it could compute from
// the form alone; whether a credential can be carried is now the provider's
// answer, and it arrives at launch.
func TestTheInventoryPreviewNamesTheFileItWillWrite(t *testing.T) {
	c := sshConnector()
	f := newForm(c)
	fieldByLabel(t, f, "user@host").SetValue("chris@10.0.0.4")
	fieldByLabel(t, f, "password").SetValue(sentinel)

	got := (launchRequest{form: f}).preview(c, scanAction())
	want := "cnspec scan --inventory-file <generated, 0600>"
	if got != want {
		t.Errorf("bar = %q, want %q", got, want)
	}
	// The connector's name is absent on purpose: the inventory carries the
	// connection type, and naming a target as well makes cnspec parse one it
	// then discards. Verified live -- `cnspec scan local --inventory-file <ssh
	// asset>` scans the ssh asset and never touches this machine.
	if strings.Contains(got, " ssh ") {
		t.Errorf("the bar names a target beside the inventory file: %q", got)
	}
}
