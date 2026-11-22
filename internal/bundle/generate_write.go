// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
)

// This is the write half of the `policy generate` seam: turning an accepted
// check into bytes in a bundle file. It lives here rather than beside a command
// because there are two front ends over it -- the line-oriented wizard and the
// launcher's authoring pane -- and `apps/cnspec/cmd` imports the launcher, so
// the launcher cannot import it back.

// ErrUIDExists reports a uid already present in the bundle. Callers offer a
// different uid rather than overwriting a check the user did not name.
var ErrUIDExists = errors.New("uid already used in this bundle")

// GeneratedGroupTitle names the group generated checks are appended to.
const GeneratedGroupTitle = "Generated checks"

// LoadFile reads a bundle, treating a missing file as an empty one so a check
// can be written into a path that does not exist yet.
func LoadFile(file string) (*Bundle, error) {
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return &Bundle{}, nil
	}
	if err != nil {
		return nil, err
	}
	b, err := ParseYaml(data)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse %s", file)
	}
	return b, nil
}

// AppendCheck adds a new check to a bundle file (creating it if absent),
// preserving existing formatting/comments and writing atomically. provider, when
// known, is added to the policy's require list (see AddCheck).
func AppendCheck(file, uid, title, desc, filter, mql, provider string) error {
	b, err := LoadFile(file)
	if err != nil {
		return err
	}
	if err := AddCheck(b, PolicyUIDForFile(file), uid, title, desc, filter, mql, provider); err != nil {
		return err
	}
	out, err := FormatBundle(b, false)
	if err != nil {
		return err
	}
	return writeFileAtomic(file, out)
}

// AddCheck places a check inside a policy group.
//
// A check that lives only in the top-level `queries:` block is not scannable:
// lint reports it as query-unassigned and `cnspec scan --policy-bundle` then
// fails with "a policy or framework mrn is required". Both front ends tell the
// user to lint and scan what was written, so what is written has to survive
// both.
//
// provider, when not empty, is added to the policy's `require:` list. Lint warns
// on a policy without one (policy-missing-require), and it is what lets
// `cnspec scan --policy-bundle` install the provider the check needs.
func AddCheck(b *Bundle, policyUID, uid, title, desc, filter, mql, provider string) error {
	if b == nil {
		return errors.New("no bundle to add the check to")
	}
	if strings.TrimSpace(uid) == "" {
		return errors.New("a check needs a uid")
	}
	if QueryUIDs(b)[uid] {
		return errors.Wrapf(ErrUIDExists, "check %q", uid)
	}

	q := &Mquery{Uid: uid, Title: title, Mql: mql}
	if strings.TrimSpace(filter) != "" {
		q.Filters = &Filters{Items: map[string]*Mquery{"": {Mql: filter}}}
	}
	if strings.TrimSpace(desc) != "" {
		q.Docs = &MqueryDocs{Desc: desc}
	}

	policy := targetPolicy(b, policyUID)
	group := generatedGroup(policy)
	group.Checks = append(group.Checks, q)
	CompletePolicy(policy, provider, filter)
	return nil
}

// Policy tags lint requires on every policy (policy-required-tags-missing).
const (
	CategoryTag = "mondoo.com/category"
	PlatformTag = "mondoo.com/platform"
)

// defaultCategory is what 74 of the 81 policies in content/ declare; the rest
// are best-practices. A generated check is written from a requirement someone
// wants enforced, which is the security case.
const defaultCategory = "security"

// CompletePolicy fills in the policy metadata a check implies and lint asks for:
// the check's provider in `require:` (policy-missing-require, and what lets a
// scan install the provider), and the mondoo.com/category and
// mondoo.com/platform tags (policy-required-tags-missing). A value the author
// set is never changed, so a policy covering several platforms keeps the
// platform tag it already has. It returns one line per addition, for the
// caller to report.
func CompletePolicy(p *Policy, provider, filter string) []string {
	if p == nil {
		return nil
	}
	var added []string
	provider = strings.TrimSpace(provider)
	if provider != "" && !requiresProvider(p, provider) {
		p.Require = append(p.Require, &Requirement{Provider: provider})
		added = append(added, "require provider "+provider)
	}
	if p.Tags == nil {
		p.Tags = map[string]string{}
	}
	if _, ok := p.Tags[CategoryTag]; !ok {
		p.Tags[CategoryTag] = defaultCategory
		added = append(added, "tag "+CategoryTag+": "+defaultCategory)
	}
	if _, ok := p.Tags[PlatformTag]; !ok {
		if platform := PlatformTagFor(provider, filter); platform != "" {
			p.Tags[PlatformTag] = platform
			added = append(added, "tag "+PlatformTag+": "+platform)
		}
	}
	return added
}

func requiresProvider(p *Policy, provider string) bool {
	for _, r := range p.Require {
		if r != nil && r.Provider == provider {
			return true
		}
	}
	return false
}

// platformTags maps a provider to the mondoo.com/platform value the policies
// in content/ use for it. Cloud and SaaS providers carry a class after the
// platform ("aws,cloud", "okta-org,saas"), which is how the catalog groups
// them. A provider not listed is tagged with its own name.
var platformTags = map[string]string{
	"aws": "aws,cloud", "gcp": "gcp,cloud", "azure": "azure,cloud",
	"oci": "oci,cloud", "alicloud": "alicloud,cloud", "digitalocean": "digitalocean,cloud",
	"hetzner": "hetzner,cloud", "openstack": "openstack,cloud", "stackit": "stackit,cloud",
	"k8s":              "kubernetes",
	"ms365":            "microsoft365,saas",
	"okta":             "okta-org,saas",
	"slack":            "slack-team,saas",
	"google-workspace": "google-workspace,saas",
	"cloudflare":       "cloudflare,saas", "snowflake": "snowflake,saas", "databricks": "databricks,saas",
	"mongodbatlas": "mongodbatlas,saas", "grafana": "grafana,saas", "tailscale": "tailscale,saas",
	"vercel": "vercel,saas", "shodan": "shodan,saas",
	"atlassian": "atlassian-admin,atlassian-jira,atlassian-confluence,saas",
	"vsphere":   "vmware,vmware-vsphere",
	"arista":    "arista-eos,networkdevice",
	"network":   "host",
}

// PlatformTagFor returns the mondoo.com/platform value for a check's provider.
// The os provider covers several operating systems, so its value comes from
// the check's asset filter, the same signal a scan selects assets by; an os
// check filtered to nothing in particular is tagged linux, the family
// DefaultFilter proposes for it. An unknown provider gives "".
func PlatformTagFor(provider, filter string) string {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return ""
	}
	if provider == "os" {
		f := strings.ToLower(filter)
		var platforms []string
		for _, name := range []string{"windows", "macos", "linux"} {
			if strings.Contains(f, `"`+name+`"`) || (name == "macos" && strings.Contains(f, `"darwin"`)) {
				platforms = append(platforms, name)
			}
		}
		if len(platforms) == 0 {
			return "linux"
		}
		return strings.Join(platforms, ",")
	}
	if tag, ok := platformTags[provider]; ok {
		return tag
	}
	return provider
}

// PoliciesWithCheck returns the policies whose groups list a check with this
// uid.
func PoliciesWithCheck(b *Bundle, uid string) []*Policy {
	if b == nil || uid == "" {
		return nil
	}
	var out []*Policy
	for _, p := range b.Policies {
		if p == nil {
			continue
		}
	groups:
		for _, g := range p.Groups {
			if g == nil {
				continue
			}
			for _, c := range g.Checks {
				if c != nil && c.Uid == uid {
					out = append(out, p)
					break groups
				}
			}
		}
	}
	return out
}

// targetPolicy returns the policy new checks join: the bundle's first policy
// when it has one, otherwise a new minimal policy (uid + name + semver version
// are what `cnspec policy lint` requires).
func targetPolicy(b *Bundle, policyUID string) *Policy {
	if len(b.Policies) > 0 && b.Policies[0] != nil {
		return b.Policies[0]
	}
	p := &Policy{
		Uid:     policyUID,
		Name:    humanize(policyUID),
		Version: "1.0.0",
	}
	b.Policies = append(b.Policies, p)
	return p
}

// generatedGroup returns the group new checks are appended to, creating it when
// needed.
//
// It is deliberately a group of its own rather than the policy's first group: a
// group-level filter gates every check inside it, so dropping a fresh check into
// an existing group can silently restrict it to assets it was never written for
// (a group filtered to hosts running kube-apiserver, say). The checks carry
// their own filters, which is what policy-missing-asset-filter asks for.
func generatedGroup(p *Policy) *PolicyGroup {
	for _, g := range p.Groups {
		if g != nil && g.Title == GeneratedGroupTitle && (g.Filters == nil || len(g.Filters.Items) == 0) {
			return g
		}
	}
	g := &PolicyGroup{Title: GeneratedGroupTitle}
	p.Groups = append(p.Groups, g)
	return g
}

// policyUidRe mirrors the uid requirement `cnspec policy lint` enforces on a
// policy (bundle-invalid-uid): 4-100 characters of lowercase letters, digits,
// dashes and underscores.
var policyUidRe = regexp.MustCompile(`^([\d\-_]|[a-z]){4,100}$`)

// PolicyUIDForFile derives the uid of a generated policy from the bundle's file
// name, falling back to a generic uid when that would not lint.
func PolicyUIDForFile(file string) string {
	base := filepath.Base(file)
	base = strings.TrimSuffix(base, ".yaml")
	base = strings.TrimSuffix(base, ".yml")
	base = strings.TrimSuffix(base, ".mql")
	uid := Slugify(base)
	if !policyUidRe.MatchString(uid) {
		return "generated-policy"
	}
	return uid
}

// maxCheckUIDLen leaves room under the 200-character limit of queryUid for the
// "-N" suffix NextFreeUID appends on a collision.
const maxCheckUIDLen = 190

// CheckUIDForTitle derives a check uid from its title that passes lint's query
// uid rule (5-200 characters). Slugify alone does not: a short title ("SSH")
// gives a uid that is too short, a long one gives one that is too long, and a
// title with no ASCII letters or digits gives an empty uid, which AppendCheck
// refuses. A short slug is padded rather than replaced, so it still says what
// the check is about; an empty one falls back to a generic uid.
func CheckUIDForTitle(title string) string {
	uid := Slugify(title)
	if len(uid) > maxCheckUIDLen {
		uid = strings.TrimRight(uid[:maxCheckUIDLen], "-")
	}
	switch {
	case uid == "":
		return "generated-check"
	case !queryUid.MatchString(uid):
		return uid + "-check"
	}
	return uid
}

// humanize turns a uid into a policy name ("aws-s3" -> "Aws S3").
func humanize(uid string) string {
	fields := strings.FieldsFunc(uid, func(r rune) bool { return r == '-' || r == '_' })
	for i, f := range fields {
		fields[i] = strings.ToUpper(f[:1]) + f[1:]
	}
	if len(fields) == 0 {
		return uid
	}
	return strings.Join(fields, " ")
}

// Slugify turns a title into a lowercase, dash-separated uid.
func Slugify(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// NextFreeUID suffixes a uid until it is free, so an offered default is one the
// user can accept rather than a collision they have to resolve by hand.
func NextFreeUID(uid string, taken map[string]bool) string {
	if uid == "" || !taken[uid] {
		return uid
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", uid, i)
		if !taken[candidate] {
			return candidate
		}
	}
}

// writeFileAtomic writes data to path via a temp file in the same directory
// followed by a rename, so an interrupted or failed write cannot truncate or
// corrupt the existing policy file. The rename is atomic on the same filesystem.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".gen-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	// flush to disk before the rename so a crash cannot leave a zero-length file
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// preserve the existing file's permissions rather than widening to 0644
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
