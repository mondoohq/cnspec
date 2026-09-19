# content/querypacks/AGENTS.md

Authoring rules for the `*.mql.yaml` query packs in this directory. Loads
alongside [`content/AGENTS.md`](../AGENTS.md), which covers MQL itself. Terse on
purpose.

36 files hold 38 packs and 853 query definitions.
`mondoo-github-inventory.mql.yaml` is the only file with more than one pack.

## A query pack collects; it never scores

`packs:` hold `queries:`, not `checks:`. A query returns data and has no verdict,
so nothing here carries `impact:`, `compliance/*` tags, `remediation:`, or
`audit:`, and `grep` confirms zero occurrences of each across the directory.
Don't import those conventions from `content/AGENTS.md`; the impact bands,
compliance-tag resolution, OWASP parity test, and IaC fixture suites all apply to
policies, not to packs.

What does carry over: the MQL traps, the ban on `—`/`–`/`--` and parenthetical
asides in prose, and the rule that Markdown in `desc` must render.

## Every query needs a title and a description

Both, on every definition. This is the rule that gets forgotten.

```yaml
- uid: mondoo-asset-inventory-aws-ec2-instances-exposure
  title: EC2 instance internet exposure
  docs:
    desc: Whether each instance is internet reachable, and why.
```

- **60 characters or fewer, one line.** The description is what the UI shows in
  the query list, next to the title. Longer text is truncated there, so a
  description that needs a click to be read has failed at its job.
- **Say what the data is, not what the query does.** "Listening ports with
  protocol, address, and process", not "Lists all listening network ports with
  their associated processes to surface network-facing services". Drop
  "Returns", "Lists all", and the trailing "so that an investigator can ..."
  clause; at this length they cost more than they explain.
- **Don't restate the title.** The two are shown together. A title of
  `Installed packages` and a description of "Lists installed packages" wastes the
  line; "Installed packages with version, arch, and origin" earns it.
- **No MQL, no field lists copied from the query, no fenced code blocks.** The
  query body is already on screen.

### When a second paragraph is allowed

Only when the extra text records something that changes how a reader interprets
the result and that nothing else in the pack states: a field that is always null
on this platform, a privilege the query needs, a scope it silently excludes, a
cross-query join. Twenty-eight queries qualify today, for example:

```yaml
desc: |
  Listening ports with protocol, address, and state.

  AIX reports sockets through netstat, which names no owning process, so
  `user` and `process` are null on this platform.
```

The first line still has to stand alone in 60 characters, because that is all
the list view shows. "Why this matters" prose, restatements of the title, and
anything recoverable from the query body do not qualify.

## A `groups[].queries` entry is usually a reference, not a definition

Most packs declare queries once under a top-level `queries:` list and then
reference them by uid from `groups[].queries`. The reference is a bare
`- uid: ...` with no sibling keys:

```yaml
    groups:
      - title: Amazon S3 Bucket
        filters: asset.runtime == "aws"
        queries:
          - uid: mondoo-asset-inventory-aws-s3-bucket-access   # reference
queries:
  - uid: mondoo-asset-inventory-aws-s3-bucket-access           # definition
    title: S3 bucket access configuration
    docs:
      desc: Public access blocks, encryption, and policy principals.
```

`docs:` written onto the reference is silently dropped; the resolved bundle takes
the definition's. So a uid appears twice and only one occurrence is the place to
edit. Anchor on the occurrence that has sibling keys at the item's key indent,
and never bulk-edit by matching `- uid:` alone: that string is also a prefix of
the six-space form inside `variants:`, and a blind `str.replace` corrupts the
file while the lint error points somewhere else.

## Variant children need their own descriptions

A variant parent holds `title` and `docs` and no `mql`; that is normal, not a
defect. Each child is listed separately once a filter selects its platform, so
each needs a description that says which scope it runs at rather than repeating
the parent's:

| Suffix | Runs on | Description says |
|---|---|---|
| `-all` / `-api` | the account, subscription, or project asset | "Every X in the account." |
| `-single` | one resource asset | "A single X asset." |
| `-legacy` / `-flexible` | Azure single vs flexible servers | which server model |

`mondoo-asset-inventory-azure-storageAccounts` and its `-api` / `-single` pair
are the reference shape.

## Pack-level fields

Every pack carries `summary:` (130 characters max, verb-first) and a `docs.desc`
that opens with one prose sentence on what the pack collects and closes with a
fenced `cnspec scan` invocation naming the file. 32 of the 38 also point at
Mondoo Platform for automatic enablement. Match the neighbours.

`tags: mondoo.com/category` is `security`, `best-practices`, or `inventory`, and
`require: - provider: <name>` names the provider the queries need.

## Validation

Lint is the only gate on this directory. There is no Go test, no fixture suite,
and no remediation validator here, so a mistake that lint accepts ships.

```bash
cnspec policy lint ./content/querypacks       # what `make test/content/lint` runs
```

Read the terminal verdict (`valid policy bundle(s)`) and the error count. **Do
not compare the warning list between runs**: it is not deterministic. Three runs
against an unmodified tree reported 6, 19, and 7 `query-deprecated-symbol`
warnings, so a warning that appears after your change is not evidence your change
caused it.

After a bulk edit, prove you changed only what you meant to by parsing both
revisions and comparing with the edited field stripped out:

```bash
python3 -c '
import subprocess, yaml, glob
def strip(o):
    if isinstance(o, dict): return {k: strip(v) for k, v in o.items() if k != "desc"}
    if isinstance(o, list): return [strip(v) for v in o]
    return o
for f in sorted(glob.glob("content/querypacks/*.mql.yaml")):
    old = subprocess.run(["git","show",f"HEAD:{f}"],capture_output=True,text=True).stdout
    if strip(yaml.safe_load(old)) != strip(yaml.safe_load(open(f))): print("CHANGED:", f)
'
```

Spelling runs repo-wide through `typos` against `typos.toml`; there is no
`expect.txt`.
