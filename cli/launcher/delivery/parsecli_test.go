// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package delivery

import (
	"testing"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers-sdk/v1/vault"
	"go.mondoo.com/mql/types"
)

// A secret is sent to the provider and left off the command line, and those are
// two different things about the same field.
//
// Args() and RequestFor() read the same form through the same visibility rule
// and disagree about exactly one row, which is the whole reason FlagFields
// exists. If they ever agree about a secret, one of two bugs has happened: the
// credential is on a command line `ps auxww` publishes, or it never reached the
// provider at all.
func TestASecretIsSentToTheProviderAndNotToArgv(t *testing.T) {
	f := sshForm()
	req := RequestFor(f, []plugin.Flag{{Long: "password", Type: plugin.FlagType_String}})

	if got := req.Flags["password"]; got == nil || string(got.Value) != "<PLACEHOLDER-not-a-real-secret>" {
		t.Fatalf("the password did not reach the provider: %+v", req.Flags)
	}
	for _, arg := range f.Args() {
		if arg == "<PLACEHOLDER-not-a-real-secret>" || arg == "--password" {
			t.Fatalf("the password reached the command line: %v", f.Args())
		}
	}
	if len(req.Args) != 1 || req.Args[0] != "chris@10.0.0.4" {
		t.Fatalf("positional arguments = %v", req.Args)
	}
}

// The connector's declaration decides a flag's type, not the widget drawn over
// it, because that is what the provider unmarshals on the other side.
//
// --discover is the one flag with no declaration at all: the CLI synthesizes it
// rather than the provider declaring it, so a zero plugin.Flag arrives and the
// widget has the last word. Getting that backwards sends a list flag a string
// and the provider reads raw protobuf bytes as text -- which is not
// hypothetical, see the activedirectory --port note in parsecli.go.
func TestFlagTypesFollowTheConnectorsDeclaration(t *testing.T) {
	mk := func(d tuiform.Decl, set func(*tuiform.Field)) tuiform.Field {
		fd := tuiform.NewField(d)
		set(&fd)
		return fd
	}
	f := tuiform.New("probe", []tuiform.Field{
		mk(tuiform.Decl{Label: "port", Flag: "port", Kind: tuiform.KindText},
			func(fd *tuiform.Field) { fd.SetValue("8443") }),
		mk(tuiform.Decl{Label: "sudo", Flag: "sudo", Kind: tuiform.KindBool},
			func(fd *tuiform.Field) { fd.SetOn(true) }),
		mk(tuiform.Decl{Label: "discover", Flag: "discover", Kind: tuiform.KindMultiChoice,
			Options: []string{"auto", "all"}},
			func(fd *tuiform.Field) { fd.TogglePick("all") }),
		mk(tuiform.Decl{Label: "repos", Flag: "repos", Kind: tuiform.KindText},
			func(fd *tuiform.Field) { fd.SetValue("one, two") }),
	})

	req := RequestFor(f, []plugin.Flag{
		{Long: "port", Type: plugin.FlagType_Int},
		{Long: "sudo", Type: plugin.FlagType_Bool},
		{Long: "repos", Type: plugin.FlagType_List},
		// discover is deliberately absent: the CLI synthesizes it.
	})

	if got := req.Flags["port"]; got == nil || got.Type != string(types.Int) {
		t.Errorf("port = %+v, want an int primitive", got)
	}
	if got := req.Flags["sudo"]; got == nil || got.Type != string(types.Bool) {
		t.Errorf("sudo = %+v, want a bool primitive", got)
	}
	if got := req.Flags["repos"]; got == nil || got.Type != string(types.Array(types.String)) {
		t.Errorf("repos = %+v, want a string array", got)
	}
	if got := req.Flags["discover"]; got == nil || got.Type != string(types.Array(types.String)) {
		t.Errorf("discover = %+v, want a string array from the multi-choice", got)
	}
}

// Where the provider put the secret is looked up rather than declared, and it
// has to be found wherever the provider chose to put it.
func TestLocateFindsTheValueWhereverItLanded(t *testing.T) {
	cases := []struct {
		name string
		conn *inventory.Config
		want Placement
	}{
		{
			name: "a password credential, the common case",
			conn: &inventory.Config{Credentials: []*vault.Credential{
				{Type: vault.CredentialType_password, Password: "<PLACEHOLDER-not-a-real-secret>"}}},
			want: PlacedCredential,
		},
		{
			// clickhousecloud: the api key is the user of the pair whose
			// password is the api secret.
			name: "the credential's user, because the key is the identity",
			conn: &inventory.Config{Credentials: []*vault.Credential{
				{Type: vault.CredentialType_password, User: "<PLACEHOLDER-not-a-real-secret>"}}},
			want: PlacedCredential,
		},
		{
			name: "a bearer token in Secret",
			conn: &inventory.Config{Credentials: []*vault.Credential{
				{Type: vault.CredentialType_bearer, Secret: []byte("<PLACEHOLDER-not-a-real-secret>")}}},
			want: PlacedCredential,
		},
		{
			// openai, ollama, huggingface and claude never read Credentials.
			name: "a connection option, which no vault reference reaches",
			conn: &inventory.Config{Options: map[string]string{"api-key": "<PLACEHOLDER-not-a-real-secret>"}},
			want: PlacedOption,
		},
		{
			// alicloud --sts-token, verified against the installed provider.
			name: "nowhere, because the provider dropped it",
			conn: &inventory.Config{Options: map[string]string{"region": "eu-central-1"}},
			want: PlacedNowhere,
		},
		{
			// A provider free to echo the value into an option as well is
			// still keychain-protectable, and the credential is the better
			// answer of the two.
			name: "both, and the credential wins",
			conn: &inventory.Config{
				Options:     map[string]string{"password": "<PLACEHOLDER-not-a-real-secret>"},
				Credentials: []*vault.Credential{{Password: "<PLACEHOLDER-not-a-real-secret>"}},
			},
			want: PlacedCredential,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			placed := Locate(sshForm(), &inventory.Asset{Connections: []*inventory.Config{tc.conn}})
			if len(placed) != 1 {
				t.Fatalf("want one located secret, got %d", len(placed))
			}
			if placed[0].Placement != tc.want {
				t.Errorf("placement = %v, want %v", placed[0].Placement, tc.want)
			}
		})
	}
}

// A field that names a file holding a credential is not a credential, so it is
// not looked for and cannot make a launch refuse.
//
// The provider is free to open the file and never carry the path at all, which
// is what okta does with --private-key. Treating the path as a lost secret
// would refuse every one of those launches.
func TestAReferenceFieldIsNotLookedFor(t *testing.T) {
	key := tuiform.NewField(tuiform.Decl{
		Label: "identity-file", Flag: "identity-file", Kind: tuiform.KindText,
		Secret: true, Reference: true, Section: tuiform.SectionCredential,
	})
	key.SetValue("~/.ssh/id_ed25519")
	f := sshForm()
	f.SetFields(append(f.Fields(), key))

	placed := Locate(f, sshAsset())
	if len(placed) != 1 || placed[0].Flag != "password" {
		t.Fatalf("want only the password located, got %+v", placed)
	}
}

// A form with no credential does not need a file, and one with a credential
// does. That is the whole of the route decision now.
func TestRouteForDependsOnlyOnWhetherASecretWasTyped(t *testing.T) {
	if got := RouteFor(tuiform.New("local", nil)); got != Plain {
		t.Errorf("an empty form routes as %v, want Plain", got)
	}
	if got := RouteFor(sshForm()); got != Inventory {
		t.Errorf("a form holding a password routes as %v, want Inventory", got)
	}

	// The reversal worth asserting: a typed password used to be discarded in
	// favour of --ask-pass, so the child prompted for a credential the user had
	// already given it. Ticking the toggle is still available and is still the
	// Plain route; typing is no longer silently converted into it.
	ask := tuiform.NewField(tuiform.Decl{
		Label: "ask-pass", Flag: "ask-pass", Kind: tuiform.KindBool,
		Section: tuiform.SectionCredential,
	})
	ask.SetOn(true)
	withAsk := sshForm()
	withAsk.SetFields(append(withAsk.Fields(), ask))
	if got := RouteFor(withAsk); got != Inventory {
		t.Errorf("a typed password beside --ask-pass routes as %v, want Inventory", got)
	}

	noPassword := tuiform.New("ssh", []tuiform.Field{ask})
	if got := RouteFor(noPassword); got != Plain {
		t.Errorf("--ask-pass on its own routes as %v, want Plain", got)
	}
	if args := noPassword.Args(); len(args) != 1 || args[0] != "--ask-pass" {
		t.Errorf("--ask-pass did not reach the command line: %v", args)
	}
}
