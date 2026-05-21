# cnspec policy graph

Navigate and query the structure of cnspec policy bundles (`.mql.yaml` files) using graph commands. These commands build a graph of policies, groups, checks, frameworks, controls, and their relationships, then let you explore it without reading thousands of lines of YAML.

## Quick start

```bash
# Build cnspec (or use an installed binary)
make cnspec/build

# See what's in a policy bundle
cnspec policy graph export ./content/mondoo-linux-security.mql.yaml --format json

# What does a policy contain?
cnspec policy graph callees mondoo-linux-security ./content/mondoo-linux-security.mql.yaml

# What references a specific check?
cnspec policy graph callers mondoo-linux-security-ssh-root-login-is-disabled ./content/

# Full context with YAML snippets
cnspec policy graph context mondoo-linux-security-ssh-root-login-is-disabled ./content/ --depth 2
```

## Commands

| Command | Purpose |
|---------|---------|
| `cnspec policy graph search <query> <path>` | Find nodes by name, title, or UID |
| `cnspec policy graph callers <uid> <path>` | What references this node (inbound edges) |
| `cnspec policy graph callees <uid> <path>` | What this node contains/references (outbound edges) |
| `cnspec policy graph context <uid> <path>` | LLM-friendly context with YAML snippets |
| `cnspec policy graph paths <from> <to> <path>` | Find paths between two nodes |
| `cnspec policy graph reachable <uid> <path>` | All nodes transitively reachable |
| `cnspec policy graph export <path>` | Export the full graph |

### Flags

- `--json` — JSON output (search, callers, callees, paths, reachable)
- `--kind` — Filter by node kind: policy, check, group, query, framework, control (search only)
- `--tag` — Filter by tag key (search only)
- `--impact N` — Minimum impact score (search only)
- `--limit N` — Maximum results, default 50 (search only)
- `--depth N` — Neighborhood depth for context (default 2)
- `--format json|dot` — Export format (export command only)

### Path argument

The `<path>` argument can be:

- A directory: walks for all `.mql.yaml` files recursively
- One or more specific `.mql.yaml` files
- Multiple paths: `cnspec policy graph callers check-uid file1.mql.yaml file2.mql.yaml`

## Graph concepts

### Node types

| Kind | Description |
|------|-------------|
| `policy` | A security policy (e.g., `mondoo-linux-security`) |
| `group` | A group of checks within a policy (e.g., "SSH Configuration") |
| `check` | A scoring query with MQL code and impact rating |
| `query` | A data-only query (no scoring) |
| `framework` | A compliance framework (e.g., CIS Benchmark, ISO 27001) |
| `control` | A control within a framework (e.g., CIS 5.2.1) |
| `framework_map` | Maps framework controls to policy checks |

### Edge types

| Kind | Description |
|------|-------------|
| `contains` | Structural parent-child (policy -> group -> check, framework -> control) |
| `maps_to` | Compliance mapping (control -> check/policy via framework_map) |
| `depends_on` | Policy/framework dependencies |
| `variant_of` | Check variant relationship (parent -> platform-specific variant) |

## Examples

### Search for nodes

```bash
# Find all SSH-related checks
$ cnspec policy graph search ssh ./content/ --kind check
```

```
check        check:mondoo-linux-security-ssh-root-login-is-d... Ensure SSH root login is disabled or ... (content/mondoo-linux-security.mql.yaml:10822)
check        check:mondoo-linux-security-ssh-x11-forwarding-... Ensure SSH X11 forwarding is disabled    (content/mondoo-linux-security.mql.yaml:10363)
...
```

```bash
# Find checks by title (searches titles, not just UIDs)
$ cnspec policy graph search "root login" ./content/ --kind check
```

```
check        check:mondoo-junos-security-ssh-root-login-disa... Ensure root login via SSH is denied      (content/mondoo-junos-security.mql.yaml:101)
check        check:mondoo-freebsd-security-ssh-root-login-is... Ensure SSH root login is disabled        (content/mondoo-freebsd-security.mql.yaml:2458)
check        check:mondoo-linux-security-ssh-root-login-is-d... Ensure SSH root login is disabled or ... (content/mondoo-linux-security.mql.yaml:10822)
...
```

```bash
# List all policies
$ cnspec policy graph search "" ./content/ --kind policy
```

```
policy       policy:mondoo-linux-security                       Mondoo Linux Security                    (content/mondoo-linux-security.mql.yaml:4)
policy       policy:mondoo-aws-security                         Mondoo AWS Security                      (content/mondoo-aws-security.mql.yaml:4)
...
```

```bash
# JSON output for programmatic use
$ cnspec policy graph search ssh ./content/ --kind check --json --limit 2
```

### Get an overview of a policy bundle

```bash
$ cnspec policy graph export ./content/mondoo-linux-security.mql.yaml --format json | \
    python3 -c "
import json, sys, collections
d = json.load(sys.stdin)
kinds = collections.Counter(n['kind'] for n in d['nodes'])
print(f'Nodes: {len(d[\"nodes\"])}, Edges: {len(d[\"edges\"])}')
for k, v in kinds.most_common(): print(f'  {k}: {v}')
"
```

```
Nodes: 130, Edges: 129
  check: 119
  group: 10
  policy: 1
```

### Explore policy structure top-down

Start by seeing what a policy contains:

```bash
$ cnspec policy graph callees mondoo-linux-security ./content/mondoo-linux-security.mql.yaml
```

```
policy:mondoo-linux-security contains/references:
  [contains] group:mondoo-linux-security-group-0 (content/mondoo-linux-security.mql.yaml:37)
  [contains] group:mondoo-linux-security-group-1 (content/mondoo-linux-security.mql.yaml:47)
  [contains] group:mondoo-linux-security-group-2 (content/mondoo-linux-security.mql.yaml:59)
  [contains] group:mondoo-linux-security-group-3 (content/mondoo-linux-security.mql.yaml:69)
  [contains] group:mondoo-linux-security-group-4 (content/mondoo-linux-security.mql.yaml:84)
  [contains] group:mondoo-linux-security-group-5 (content/mondoo-linux-security.mql.yaml:96)
  [contains] group:mondoo-linux-security-group-6 (content/mondoo-linux-security.mql.yaml:118)
  [contains] group:mondoo-linux-security-group-7 (content/mondoo-linux-security.mql.yaml:142)
  [contains] group:mondoo-linux-security-group-8 (content/mondoo-linux-security.mql.yaml:173)
  [contains] group:mondoo-linux-security-group-9 (content/mondoo-linux-security.mql.yaml:180)
```

Then drill into a specific group to see its checks:

```bash
$ cnspec policy graph callees mondoo-linux-security-group-6 ./content/mondoo-linux-security.mql.yaml
```

```
group:mondoo-linux-security-group-6 contains/references:
  [contains] check:mondoo-linux-security-only-strong-ciphers-are-used (content/mondoo-linux-security.mql.yaml:11173)
  [contains] check:mondoo-linux-security-only-strong-kex-algorithms-are-used (content/mondoo-linux-security.mql.yaml:11398)
  [contains] check:mondoo-linux-security-only-strong-mac-algorithms-are-used (content/mondoo-linux-security.mql.yaml:11287)
  [contains] check:mondoo-linux-security-permissions-on-etcsshsshd-config-are-configured (content/mondoo-linux-security.mql.yaml:9061)
  [contains] check:mondoo-linux-security-permissions-on-ssh-private-host-key-files-are-configured (content/mondoo-linux-security.mql.yaml:9917)
  [contains] check:mondoo-linux-security-permissions-on-ssh-public-host-key-files-are-configured (content/mondoo-linux-security.mql.yaml:10022)
  [contains] check:mondoo-linux-security-ssh-access-is-limited (content/mondoo-linux-security.mql.yaml:11835)
  [contains] check:mondoo-linux-security-ssh-hostbasedauthentication-is-disabled (content/mondoo-linux-security.mql.yaml:10707)
  [contains] check:mondoo-linux-security-ssh-idle-timeout-interval-is-configured (content/mondoo-linux-security.mql.yaml:11596)
  [contains] check:mondoo-linux-security-ssh-ignorerhosts-is-enabled (content/mondoo-linux-security.mql.yaml:10592)
  [contains] check:mondoo-linux-security-ssh-logingracetime-is-set-to-one-minute-or-less (content/mondoo-linux-security.mql.yaml:11734)
  [contains] check:mondoo-linux-security-ssh-loglevel-is-appropriate (content/mondoo-linux-security.mql.yaml:10244)
  [contains] check:mondoo-linux-security-ssh-maxauthtries-is-set-to-4-or-less (content/mondoo-linux-security.mql.yaml:10477)
  [contains] check:mondoo-linux-security-ssh-permitemptypasswords-is-disabled (content/mondoo-linux-security.mql.yaml:10943)
  [contains] check:mondoo-linux-security-ssh-permituserenvironment-is-disabled (content/mondoo-linux-security.mql.yaml:11058)
  [contains] check:mondoo-linux-security-ssh-protocol-is-set-to-2 (content/mondoo-linux-security.mql.yaml:10125)
  [contains] check:mondoo-linux-security-ssh-root-login-is-disabled (content/mondoo-linux-security.mql.yaml:10822)
  [contains] check:mondoo-linux-security-ssh-warning-banner-is-configured (content/mondoo-linux-security.mql.yaml:11980)
  [contains] check:mondoo-linux-security-ssh-x11-forwarding-is-disabled (content/mondoo-linux-security.mql.yaml:10363)
```

### Find what references a check

```bash
$ cnspec policy graph callers mondoo-linux-security-ssh-root-login-is-disabled \
    ./content/mondoo-linux-security.mql.yaml
```

```
check:mondoo-linux-security-ssh-root-login-is-disabled is referenced by:
  [contains] group:mondoo-linux-security-group-6 (content/mondoo-linux-security.mql.yaml:118)
```

When framework bundles are loaded, this also shows compliance control mappings via `[maps_to]` edges.

### Trace the path from policy to check

```bash
$ cnspec policy graph paths mondoo-linux-security \
    mondoo-linux-security-ssh-root-login-is-disabled \
    ./content/mondoo-linux-security.mql.yaml
```

```
Path 1:
  policy:mondoo-linux-security
  → group:mondoo-linux-security-group-6
  → check:mondoo-linux-security-ssh-root-login-is-disabled
```

### Find all SSH-related checks

```bash
$ cnspec policy graph search ssh ./content/mondoo-linux-security.mql.yaml --kind check
```

```
check        check:mondoo-linux-security-permissions-on-etcs... Ensure secure permissions on /etc/ssh... (content/mondoo-linux-security.mql.yaml:9061)
check        check:mondoo-linux-security-ssh-protocol-is-set... Ensure SSH protocol is set to 2          (content/mondoo-linux-security.mql.yaml:10125)
check        check:mondoo-linux-security-ssh-root-login-is-d... Ensure SSH root login is disabled or ... (content/mondoo-linux-security.mql.yaml:10822)
...
```

### Get full context for a check

The `context` command produces LLM-friendly markdown with YAML source snippets, compliance tags, and relationships:

```bash
$ cnspec policy graph context mondoo-linux-security-ssh-root-login-is-disabled \
    ./content/mondoo-linux-security.mql.yaml --depth 2
```

```
# Policy context for check:mondoo-linux-security-ssh-root-login-is-disabled

**Focus**: `check:mondoo-linux-security-ssh-root-login-is-disabled` (check) at content/mondoo-linux-security.mql.yaml:10822
**Title**: Ensure SSH root login is disabled or set to prohibit-password
**Impact**: 100
**Tags**: compliance/csa-cloud-controls-matrix-4=cloud-controls-matrix-4-iam-02, compliance/dora=dora-art-9, ...
**Neighborhood**: 21 nodes within 2 hops

## content/mondoo-linux-security.mql.yaml

### check:mondoo-linux-security-ssh-root-login-is-disabled (check, L10822) ← FOCUS
**Title**: Ensure SSH root login is disabled or set to prohibit-password
**Impact**: 100
Referenced by: group:mondoo-linux-security-group-6 [contains]

​```yaml
  - uid: mondoo-linux-security-ssh-root-login-is-disabled
    title: Ensure SSH root login is disabled or set to prohibit-password
    ...
​```
```

### Show all reachable nodes from a policy

```bash
$ cnspec policy graph reachable mondoo-linux-security \
    ./content/mondoo-linux-security.mql.yaml
```

```
129 nodes reachable from policy:mondoo-linux-security:
  group:mondoo-linux-security-group-0 (content/mondoo-linux-security.mql.yaml:37)
  group:mondoo-linux-security-group-1 (content/mondoo-linux-security.mql.yaml:47)
  ...
  check:mondoo-linux-security-aide-is-installed (content/mondoo-linux-security.mql.yaml:198)
  check:mondoo-linux-security-core-dumps-are-restricted (content/mondoo-linux-security.mql.yaml:543)
  ...
```

### Scan the entire content directory

Point at a directory to walk all `.mql.yaml` files:

```bash
$ cnspec policy graph export ./content/ --format json | \
    python3 -c "
import json, sys, collections
d = json.load(sys.stdin)
kinds = collections.Counter(n['kind'] for n in d['nodes'])
print(f'Nodes: {len(d[\"nodes\"])}, Edges: {len(d[\"edges\"])}')
for k, v in kinds.most_common(): print(f'  {k}: {v}')
"
```

```
Nodes: 7462, Edges: 7240
  check: 6952
  group: 452
  policy: 58
```

### JSON output

All commands except `context` and `export` support `--json` for structured output:

```bash
$ cnspec policy graph callees mondoo-linux-security ./content/mondoo-linux-security.mql.yaml --json
```

```json
[
  {
    "edge": {
      "source": "content/mondoo-linux-security.mql.yaml::policy:mondoo-linux-security",
      "target": "content/mondoo-linux-security.mql.yaml::group:mondoo-linux-security-group-0",
      "kind": "contains"
    },
    "node": {
      "id": "content/mondoo-linux-security.mql.yaml::group:mondoo-linux-security-group-0",
      "name": "mondoo-linux-security-group-0",
      "qual_name": "group:mondoo-linux-security-group-0",
      "kind": "group",
      "file": "content/mondoo-linux-security.mql.yaml",
      "line": 37,
      "column": 9,
      "title": "Core",
      "parent_id": "content/mondoo-linux-security.mql.yaml::policy:mondoo-linux-security"
    }
  },
  ...
]
```

### Generate a visual diagram

```bash
cnspec policy graph export ./content/mondoo-linux-security.mql.yaml --format dot > policy.dot
dot -Tpng policy.dot -o policy.png
```

Node colors: policies (blue), groups (lavender), checks (yellow), frameworks (green), controls (pale green).

## Framework and compliance mapping

When framework bundles are loaded alongside policy bundles, the graph includes compliance relationships:

```bash
# Load both policy and framework bundles
cnspec policy graph callers <check-uid> ./policies/ ./frameworks/

# Trace framework control → check mapping
cnspec policy graph paths <control-uid> <check-uid> ./policies/ ./frameworks/
```

Framework maps create `maps_to` edges from controls to checks, so `callers` on a check shows which compliance controls require it, and `paths` traces the full chain: framework -> control -> check.

## Claude Code skill

A [Claude Code](https://docs.anthropic.com/en/docs/claude-code) skill is included for AI-assisted policy navigation. The skill teaches Claude how to use the graph commands to answer questions about policy bundles.

### Installation

```bash
make install/skills
```

This copies the skill files to `~/.claude/`, making them available across all projects:

- `~/.claude/commands/policy-graph.md` — Slash command definition
- `~/.claude/skills/policy-graph/` — Auto-trigger skill with references

### Usage in Claude Code

**Slash command** — type `/policy-graph` followed by your question:

```
/policy-graph What SSH checks does the Linux security policy have?
/policy-graph Which compliance controls map to the root login check?
/policy-graph Show me the structure of mondoo-aws-security
```

**Auto-trigger** — the skill activates automatically when Claude detects questions about policy structure, compliance mappings, or bundle navigation. Ask naturally:

```
What checks are in the Linux security policy?
How does this framework control connect to that check?
Find all SSH-related checks in the content directory.
```

### What the skill does

When triggered, Claude uses graph commands to:

1. **Orient** — `export --format json` to understand the bundle (node counts by kind)
2. **Locate** — `export` with filtering to find specific nodes by name
3. **Navigate** — `callers`/`callees`/`paths` to explore relationships
4. **Context** — `context --depth 2` for deep investigation with YAML source snippets

The skill only reads `.mql.yaml` files — it does not scan assets, write queries, or modify bundles.
