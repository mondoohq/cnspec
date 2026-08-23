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
// preserving existing formatting/comments and writing atomically.
func AppendCheck(file, uid, title, desc, filter, mql string) error {
	b, err := LoadFile(file)
	if err != nil {
		return err
	}
	if err := AddCheck(b, PolicyUIDForFile(file), uid, title, desc, filter, mql); err != nil {
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
func AddCheck(b *Bundle, policyUID, uid, title, desc, filter, mql string) error {
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

	group := generatedGroup(targetPolicy(b, policyUID))
	group.Checks = append(group.Checks, q)
	return nil
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
