// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"slices"
	"strings"
	"testing"

	"go.mondoo.com/cnspec/cli/launcher/source"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// What the value pickers claim about connectors, confronted with the recorded
// connector metadata.
//
// The pickers are cli/launcher/source and these tests are not, because the
// snapshot is: it is built from BuildCatalog, it is 190KB of recorded provider
// declarations, and it belongs to the package that owns the catalog. What each
// test asserts is still a property of a source -- that the flag a picker fills
// exists, that the flag it deliberately does not fill really is absent -- so
// each one names the source-side test it is the other half of.

// declaredSourceIDs is the whole id namespace as the launcher names it, listed
// once so the check below is about the set rather than about any one id.
//
// It is written out rather than derived for the same reason the list in
// cli/launcher/source is: deriving it from the registry would make the test
// agree with whatever happened to be registered.
var declaredSourceIDs = []string{
	srcAWSProfile, srcKubeContext, srcSSHHost, srcDockerContainer, srcDockerImage,
	srcGCPProject, srcGCPProjectAll, srcGCPZone, srcAzureSubscription, srcOCIProfile,
	srcAlicloudProfile, srcSnowflakeConnection, srcDockerContext, srcK8sNamespace,
	srcGitHubToken, srcGitLabToken, srcSlackToken, srcCloudflareToken,
	srcDigitalOceanToken, srcHetznerToken, srcOktaToken,
	srcDiscoverK8sNamespaces, srcDiscoverGitHubRepos, srcDiscoverGitLabProjects,
	srcDiscoverGitLabGroups, srcDiscoverAzureSubscriptions, srcDiscoverGCPProjects,
	srcDiscoverAWSAccounts, srcDiscoverOCITenancy, srcDiscoverAlicloudAccounts,
	srcDiscoverDigitalOceanDBs, srcDiscoverNeonProjects, srcDiscoverNetlifySites,
	srcDiscoverVercelProjects, srcDiscoverAtlasProjects, srcDiscoverHCPProjects,
	srcDiscoverClaudeWorkspaces, srcDiscoverMSSQLDatabases, srcDiscoverMySQLDatabases,
	srcDiscoverPostgresDatabases, srcDiscoverSnowflakeDatabases,
}

// Every discovery constant names a target the connector actually declares.
//
// A discovery source asks `cnspec discover <connector> --discover <target>`, so
// a target the connector does not offer produces an empty picker with a
// plausible name -- which is the mistake the whole Source contract was written
// to stop being made one provider at a time.
func TestDiscoveryIDsNameDeclaredTargets(t *testing.T) {
	byName := snapshotByName(t)
	checked := 0
	for _, id := range declaredSourceIDs {
		rest, ok := strings.CutPrefix(id, "discover.")
		if !ok {
			continue
		}
		connector, target, ok := strings.Cut(rest, ".")
		if !ok {
			t.Errorf("%q does not name a connector and a target", id)
			continue
		}
		snap, ok := byName[connector]
		if !ok {
			t.Logf("%s: %q is not in the connector snapshot", id, connector)
			continue
		}
		checked++
		if !slices.Contains(snap.Discovery, target) {
			t.Errorf("%s: %q does not declare the discovery target %q; it declares %v",
				id, connector, target, snap.Discovery)
		}
	}
	t.Logf("checked %d discovery ids against %d snapshotted connectors", checked, len(byName))
}

// Both halves of a discovery row, checked against the recorded connector
// metadata rather than against what the flag name suggests.
//
// The first half stops a picker asking for a target the connector cannot
// enumerate. The second stops the mistake this whole class of source exists to
// avoid: a plausible-looking flag that does not take what was discovered.
// aws --filters looks like the place a discovered account goes and is not --
// see TestExcludedPairsStayExcluded in cli/launcher/source.
func TestEveryTargetIsDeclaredAndHasSomewhereToPutIt(t *testing.T) {
	byName := snapshotByName(t)
	checked := 0
	for _, d := range source.DiscoverPairs() {
		snap, ok := byName[d.Connector]
		if !ok {
			t.Logf("%s: %q is not in the connector snapshot; skipped", d.ID, d.Connector)
			continue
		}
		checked++
		if !slices.Contains(snap.Discovery, d.Target) {
			t.Errorf("%s: %q does not declare the discovery target %q; it declares %v",
				d.ID, d.Connector, d.Target, snap.Discovery)
		}
		dest := flagNamed(snap, d.Flag)
		if dest == nil {
			t.Errorf("%s: %q declares no --%s for the answer to land in", d.ID, d.Connector, d.Flag)
			continue
		}
		// The flag types are not uniform: --filters is KeyValue and takes
		// key=value pairs, so a bare discovered value put there is dropped by
		// the provider's own prefix check. Only a String flag takes what these
		// sources emit.
		if plugin.FlagType(dest.Type) != plugin.FlagType_String {
			t.Errorf("%s: --%s is %v, and these sources emit a bare value",
				d.ID, d.Flag, plugin.FlagType(dest.Type))
		}
		// Every scope flag has to exist too.
		for _, flag := range d.ScopeFlags {
			if flagNamed(snap, flag) == nil {
				t.Errorf("%s: scoped by --%s, which %q does not declare", d.ID, flag, d.Connector)
			}
		}
	}
	t.Logf("checked %d of %d discovery pairs; the rest are not in the snapshot",
		checked, len(source.DiscoverPairs()))
}

// The flags the file-backed pickers fill in have to exist, or the picker is
// decoration. The other half of TestTheFiveLocalPickersAreInstantAndEnumerated.
func TestTheFlagBackedPickersNameRealFlags(t *testing.T) {
	byName := snapshotByName(t)
	checked, skipped := 0, 0
	for connector, flag := range map[string]string{
		"oci":       "profile",
		"azure":     "subscription",
		"snowflake": "account",
	} {
		snap, ok := byName[connector]
		if !ok {
			skipped++
			t.Logf("%s is not in the connector snapshot; skipped", connector)
			continue
		}
		checked++
		if flagNamed(snap, flag) == nil {
			t.Errorf("%s declares no --%s, so its picker has nowhere to put a value", connector, flag)
		}
	}
	// snowflake's picker offers accounts rather than connection names, and the
	// absence of --connection is why. See snowflakeAccountsFrom.
	if snap, ok := byName["snowflake"]; ok && flagNamed(snap, "connection") != nil {
		t.Error("snowflake declares --connection after all; that source should offer connection names")
	}
	t.Logf("checked %d connectors, skipped %d", checked, skipped)
}

// The pickers whose value travels in the environment do so because the
// connector declares no flag for it. That absence is the whole justification,
// so it is checked rather than asserted: a flag that appeared later would mean
// the value should be travelling in it.
func TestTheEnvBackedPickersFillNoFlag(t *testing.T) {
	byName := snapshotByName(t)
	checked, skipped := 0, 0
	for _, tc := range []struct {
		connectors []string
		flag       string
		id         string
	}{
		{[]string{"alicloud"}, "profile", srcAlicloudProfile},
		{[]string{"docker", "container", "local"}, "context", srcDockerContext},
	} {
		s, ok := sourceByID(tc.id)
		if !ok {
			t.Errorf("%s is not registered", tc.id)
			continue
		}
		if s.Env == "" {
			t.Errorf("%s: no Env, but its value has no flag to travel in", tc.id)
		}
		for _, name := range tc.connectors {
			snap, ok := byName[name]
			if !ok {
				skipped++
				t.Logf("%s is not in the connector snapshot; skipped", name)
				continue
			}
			checked++
			if flagNamed(snap, tc.flag) != nil {
				t.Errorf("%s declares --%s after all; the value should travel in it, not in %s",
					name, tc.flag, s.Env)
			}
		}
	}
	t.Logf("checked %d connectors, skipped %d", checked, skipped)
}

func flagNamed(snap connectorSnapshot, long string) *flagSnippet {
	for i := range snap.Flags {
		if snap.Flags[i].Long == long {
			return &snap.Flags[i]
		}
	}
	return nil
}
