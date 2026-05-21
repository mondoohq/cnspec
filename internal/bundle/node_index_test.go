// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestIndex(t *testing.T) *NodeIndex {
	t.Helper()
	data := []byte(`
policies:
  - uid: linux-security
    name: Linux Security
    groups:
      - title: SSH Configuration
        checks:
          - uid: ssh-root-login
            title: Ensure SSH root login is disabled
            impact: 100
            tags:
              compliance/cis: "5.2.10"
            mql: |
              sshd.config.params["PermitRootLogin"] == "no"
          - uid: ssh-ciphers
            title: Ensure only strong ciphers are used
            impact: 80
            mql: |
              sshd.config.ciphers.none(_ == "3des-cbc")
          - uid: ssh-protocol
            title: Ensure SSH protocol is set to 2
            impact: 90
            mql: |
              sshd.config.params["Protocol"] == "2"
      - title: Filesystem
        checks:
          - uid: tmp-noexec
            title: Ensure noexec option on /tmp
            impact: 60
            mql: |
              mount.where(path == "/tmp").options.contains("noexec")
queries:
  - uid: ssh-root-login
    title: Ensure SSH root login is disabled
    impact: 100
    tags:
      compliance/cis: "5.2.10"
    mql: |
      sshd.config.params["PermitRootLogin"] == "no"
  - uid: ssh-ciphers
    title: Ensure only strong ciphers are used
    impact: 80
    mql: |
      sshd.config.ciphers.none(_ == "3des-cbc")
  - uid: ssh-protocol
    title: Ensure SSH protocol is set to 2
    impact: 90
    mql: |
      sshd.config.params["Protocol"] == "2"
  - uid: tmp-noexec
    title: Ensure noexec option on /tmp
    impact: 60
    mql: |
      mount.where(path == "/tmp").options.contains("noexec")
`)
	b, err := ParseYaml(data)
	require.NoError(t, err)

	g := BuildGraph(map[string]*Bundle{"test.mql.yaml": b})
	return g.BuildNodeIndex()
}

func TestNodeIndex_ExactName(t *testing.T) {
	idx := buildTestIndex(t)

	nodes := idx.ExactName("ssh-root-login")
	assert.NotEmpty(t, nodes)
	assert.Equal(t, "ssh-root-login", nodes[0].Name)

	// Case-insensitive
	nodes = idx.ExactName("SSH-Root-Login")
	assert.NotEmpty(t, nodes)
}

func TestNodeIndex_ExactQualName(t *testing.T) {
	idx := buildTestIndex(t)

	nodes := idx.ExactQualName("check:ssh-root-login")
	assert.NotEmpty(t, nodes)
	assert.Equal(t, KindCheck, nodes[0].Kind)

	nodes = idx.ExactQualName("policy:linux-security")
	assert.NotEmpty(t, nodes)
	assert.Equal(t, KindPolicy, nodes[0].Kind)
}

func TestNodeIndex_Prefix(t *testing.T) {
	idx := buildTestIndex(t)

	nodes := idx.Prefix("ssh-", 0)
	assert.GreaterOrEqual(t, len(nodes), 3, "should find at least 3 SSH checks")
	for _, n := range nodes {
		assert.True(t, len(n.Name) >= 4 && n.Name[:4] == "ssh-",
			"expected name to start with ssh-, got %s", n.Name)
	}
}

func TestNodeIndex_Substring(t *testing.T) {
	idx := buildTestIndex(t)

	// Substring in name
	nodes := idx.Substring("ciphers", 0)
	assert.NotEmpty(t, nodes)
	found := false
	for _, n := range nodes {
		if n.Name == "ssh-ciphers" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find ssh-ciphers by substring")
}

func TestNodeIndex_TitleSearch(t *testing.T) {
	idx := buildTestIndex(t)

	// "root login" is in the title but not the name
	nodes := idx.Substring("root login", 0)
	assert.NotEmpty(t, nodes, "should find node by title substring")
	found := false
	for _, n := range nodes {
		if n.Name == "ssh-root-login" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find ssh-root-login by title 'root login'")
}

func TestNodeIndex_SearchCascade(t *testing.T) {
	idx := buildTestIndex(t)

	// Exact name match should return only that node, not substring matches
	nodes := idx.Search("ssh-root-login", SearchOpts{})
	assert.NotEmpty(t, nodes)
	assert.Equal(t, "ssh-root-login", nodes[0].Name)
}

func TestNodeIndex_KindFilter(t *testing.T) {
	idx := buildTestIndex(t)

	// Without filter: should find policy AND checks matching "linux"
	all := idx.Search("linux", SearchOpts{})
	assert.NotEmpty(t, all)

	// With kind filter: only policy
	policies := idx.Search("linux", SearchOpts{Kind: KindPolicy})
	assert.NotEmpty(t, policies)
	for _, n := range policies {
		assert.Equal(t, KindPolicy, n.Kind)
	}
}

func TestNodeIndex_TagFilter(t *testing.T) {
	idx := buildTestIndex(t)

	// Only ssh-root-login has the compliance/cis tag
	nodes := idx.Search("ssh", SearchOpts{TagKey: "compliance/cis"})
	assert.NotEmpty(t, nodes)
	for _, n := range nodes {
		_, ok := n.Tags["compliance/cis"]
		assert.True(t, ok, "expected node %s to have compliance/cis tag", n.Name)
	}
}

func TestNodeIndex_Limit(t *testing.T) {
	idx := buildTestIndex(t)

	nodes := idx.Search("ssh", SearchOpts{Limit: 1})
	assert.Len(t, nodes, 1)
}

func TestNodeIndex_EmptyQueryWithKind(t *testing.T) {
	idx := buildTestIndex(t)

	policies := idx.Search("", SearchOpts{Kind: KindPolicy})
	assert.NotEmpty(t, policies)
	for _, n := range policies {
		assert.Equal(t, KindPolicy, n.Kind)
	}
}

func BenchmarkNodeIndex_Build(b *testing.B) {
	data := []byte(`
policies:
  - uid: bench-policy
    name: Benchmark Policy
    groups:
      - title: Group 1
        checks:
          - uid: check-001
            title: Check one
            impact: 80
            mql: "true"
          - uid: check-002
            title: Check two
            impact: 60
            mql: "true"
          - uid: check-003
            title: Check three
            impact: 90
            mql: "true"
queries:
  - uid: check-001
    title: Check one
    impact: 80
    mql: "true"
  - uid: check-002
    title: Check two
    impact: 60
    mql: "true"
  - uid: check-003
    title: Check three
    impact: 90
    mql: "true"
`)
	bun, _ := ParseYaml(data)
	g := BuildGraph(map[string]*Bundle{"bench.mql.yaml": bun})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = BuildNodeIndex(g)
	}
}

func BenchmarkNodeIndex_Search(b *testing.B) {
	data := []byte(`
policies:
  - uid: bench-policy
    name: Benchmark Policy
    groups:
      - title: Group 1
        checks:
          - uid: check-001
            title: Check one
            impact: 80
            mql: "true"
          - uid: check-002
            title: Check two
            impact: 60
            mql: "true"
          - uid: check-003
            title: Check three
            impact: 90
            mql: "true"
queries:
  - uid: check-001
    title: Check one
    impact: 80
    mql: "true"
  - uid: check-002
    title: Check two
    impact: 60
    mql: "true"
  - uid: check-003
    title: Check three
    impact: 90
    mql: "true"
`)
	bun, _ := ParseYaml(data)
	g := BuildGraph(map[string]*Bundle{"bench.mql.yaml": bun})
	idx := BuildNodeIndex(g)

	b.Run("ExactName", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = idx.ExactName("check-001")
		}
	})
	b.Run("Prefix", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = idx.Prefix("check-", 50)
		}
	})
	b.Run("Substring", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = idx.Substring("check", 50)
		}
	})
	b.Run("SearchCascade", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = idx.Search("check-001", SearchOpts{})
		}
	})
}
