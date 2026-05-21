// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGraph_Policies(t *testing.T) {
	data := []byte(`
policies:
  - uid: linux-security
    name: Linux Security
    groups:
      - title: SSH Configuration
        checks:
          - uid: sshd-ciphers
            title: Ensure strong ciphers
            impact: 80
            mql: |
              sshd.config.ciphers.none(_ == "3des-cbc")
        queries:
          - uid: sshd-config-data
            title: SSH config data
            mql: |
              sshd.config { * }
queries:
  - uid: sshd-ciphers
    title: Ensure strong ciphers
    impact: 80
    mql: |
      sshd.config.ciphers.none(_ == "3des-cbc")
  - uid: sshd-config-data
    title: SSH config data
    mql: |
      sshd.config { * }
`)
	b, err := ParseYaml(data)
	require.NoError(t, err)

	g := BuildGraph(map[string]*Bundle{"test.mql.yaml": b})

	// Should have: 1 policy, 1 group, 2 checks/queries (from top-level queries)
	policyNodes := findByKind(g, KindPolicy)
	assert.Len(t, policyNodes, 1)
	assert.Equal(t, "linux-security", policyNodes[0].Name)

	groupNodes := findByKind(g, KindGroup)
	assert.Len(t, groupNodes, 1)
	assert.Equal(t, "SSH Configuration", groupNodes[0].Title)

	checkNodes := findByKind(g, KindCheck)
	assert.GreaterOrEqual(t, len(checkNodes), 1)

	// Policy -> Group containment edge
	pID := policyNodes[0].ID
	outEdges := g.OutEdges(pID, EdgeContains)
	assert.Len(t, outEdges, 1)
	assert.Equal(t, groupNodes[0].ID, outEdges[0].Target)

	// Group -> Check containment edge (resolved)
	gID := groupNodes[0].ID
	groupOut := g.OutEdges(gID, EdgeContains)
	assert.GreaterOrEqual(t, len(groupOut), 1)
}

func TestBuildGraph_Frameworks(t *testing.T) {
	data := []byte(`
frameworks:
  - uid: cis-benchmark
    name: CIS Benchmark
    groups:
      - title: Access Control
        controls:
          - uid: cis-1.1
            title: Ensure SSH is configured
          - uid: cis-1.2
            title: Ensure passwords are strong
`)
	b, err := ParseYaml(data)
	require.NoError(t, err)

	g := BuildGraph(map[string]*Bundle{"fw.mql.yaml": b})

	fwNodes := findByKind(g, KindFramework)
	assert.Len(t, fwNodes, 1)
	assert.Equal(t, "CIS Benchmark", fwNodes[0].Title)

	ctrlNodes := findByKind(g, KindControl)
	assert.Len(t, ctrlNodes, 2)

	// Framework -> Control containment
	fwID := fwNodes[0].ID
	outEdges := g.OutEdges(fwID, EdgeContains)
	assert.Len(t, outEdges, 2)
}

func TestBuildGraph_FrameworkMaps(t *testing.T) {
	data := []byte(`
queries:
  - uid: check-ssh
    title: Check SSH
    impact: 80
    mql: sshd.config.ciphers.length > 0
frameworks:
  - uid: cis-benchmark
    name: CIS Benchmark
    groups:
      - title: Access Control
        controls:
          - uid: cis-1.1
            title: Ensure SSH is configured
framework_maps:
  - uid: cis-to-policy
    framework_owner:
      uid: cis-benchmark
    controls:
      - uid: cis-1.1
        checks:
          - uid: check-ssh
`)
	b, err := ParseYaml(data)
	require.NoError(t, err)

	g := BuildGraph(map[string]*Bundle{"mapped.mql.yaml": b})

	// Control -> Check maps_to edge should be resolved
	ctrlNodes := findByKind(g, KindControl)
	require.Len(t, ctrlNodes, 1)

	checkNodes := findByKind(g, KindCheck)
	require.Len(t, checkNodes, 1)

	outEdges := g.OutEdges(ctrlNodes[0].ID, EdgeMapsTo)
	assert.Len(t, outEdges, 1)
	assert.Equal(t, checkNodes[0].ID, outEdges[0].Target)
}

func TestBuildGraph_Variants(t *testing.T) {
	data := []byte(`
queries:
  - uid: parent-check
    title: Parent Check
    variants:
      - uid: parent-check-aws
      - uid: parent-check-gcp
  - uid: parent-check-aws
    title: AWS variant
    mql: aws.ec2.instances { tags["Name"] != empty }
  - uid: parent-check-gcp
    title: GCP variant
    mql: gcp.compute.instances { name != empty }
`)
	b, err := ParseYaml(data)
	require.NoError(t, err)

	g := BuildGraph(map[string]*Bundle{"variants.mql.yaml": b})

	nodes := g.FindNode("parent-check")
	require.GreaterOrEqual(t, len(nodes), 1)

	parentID := ""
	for _, n := range nodes {
		if n.Name == "parent-check" {
			parentID = n.ID
			break
		}
	}
	require.NotEmpty(t, parentID)

	variantEdges := g.OutEdges(parentID, EdgeVariantOf)
	assert.Len(t, variantEdges, 2)
}

func TestBuildGraph_CrossFileResolve(t *testing.T) {
	checks := []byte(`
queries:
  - uid: my-check
    title: My Check
    impact: 90
    mql: users.list.length > 0
`)
	policies := []byte(`
policies:
  - uid: my-policy
    name: My Policy
    groups:
      - title: Group 1
        checks:
          - uid: my-check
`)
	bChecks, err := ParseYaml(checks)
	require.NoError(t, err)
	bPolicies, err := ParseYaml(policies)
	require.NoError(t, err)

	g := BuildGraph(map[string]*Bundle{
		"checks.mql.yaml":   bChecks,
		"policies.mql.yaml": bPolicies,
	})

	// The group should have a resolved edge to the check in the other file
	groupNodes := findByKind(g, KindGroup)
	require.Len(t, groupNodes, 1)

	outEdges := g.OutEdges(groupNodes[0].ID, EdgeContains)
	resolved := false
	for _, e := range outEdges {
		if g.Node(e.Target) != nil && g.Node(e.Target).Name == "my-check" {
			resolved = true
		}
	}
	assert.True(t, resolved, "cross-file check reference should be resolved")
}

func TestPolicyGraph_FindPaths(t *testing.T) {
	data := []byte(`
queries:
  - uid: check-a
    title: Check A
    impact: 80
    mql: "true"
frameworks:
  - uid: fw
    name: Framework
    groups:
      - title: Controls
        controls:
          - uid: ctrl-1
            title: Control 1
framework_maps:
  - framework_owner:
      uid: fw
    controls:
      - uid: ctrl-1
        checks:
          - uid: check-a
`)
	b, err := ParseYaml(data)
	require.NoError(t, err)

	g := BuildGraph(map[string]*Bundle{"paths.mql.yaml": b})

	ctrlNodes := findByKind(g, KindControl)
	require.Len(t, ctrlNodes, 1)
	checkNodes := findByKind(g, KindCheck)
	require.Len(t, checkNodes, 1)

	paths := g.FindPaths(ctrlNodes[0].ID, checkNodes[0].ID, 5)
	assert.GreaterOrEqual(t, len(paths), 1)
}

func TestPolicyGraph_Reachable(t *testing.T) {
	data := []byte(`
queries:
  - uid: check-1
    title: Check 1
    impact: 80
    mql: "true"
  - uid: check-2
    title: Check 2
    impact: 70
    mql: "true"
policies:
  - uid: pol
    name: Policy
    groups:
      - title: Group 1
        checks:
          - uid: check-1
          - uid: check-2
`)
	b, err := ParseYaml(data)
	require.NoError(t, err)

	g := BuildGraph(map[string]*Bundle{"reach.mql.yaml": b})

	polNodes := findByKind(g, KindPolicy)
	require.Len(t, polNodes, 1)

	reachable := g.Reachable(polNodes[0].ID)
	assert.GreaterOrEqual(t, len(reachable), 2) // at least group + checks
}

func TestPolicyGraph_WriteContext(t *testing.T) {
	data := []byte(`
queries:
  - uid: check-x
    title: Important Check
    impact: 90
    mql: |
      users.where(name == "root").list { shell != "/bin/bash" }
policies:
  - uid: security-pol
    name: Security Policy
    groups:
      - title: User Checks
        checks:
          - uid: check-x
`)
	b, err := ParseYaml(data)
	require.NoError(t, err)

	g := BuildGraph(map[string]*Bundle{"ctx.mql.yaml": b})

	checkNodes := findByKind(g, KindCheck)
	require.GreaterOrEqual(t, len(checkNodes), 1)

	var buf bytes.Buffer
	err = g.WriteContext(&buf, checkNodes[0].ID, 2, "")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Policy context for")
	assert.Contains(t, output, "check-x")
	assert.Contains(t, output, "Important Check")
	assert.Contains(t, output, "Impact")
	assert.Contains(t, output, "FOCUS")
}

func findByKind(g *PolicyGraph, kind NodeKind) []*GraphNode {
	var result []*GraphNode
	for _, n := range g.Nodes {
		if n.Kind == kind {
			result = append(result, n)
		}
	}
	return result
}
