---
name: mql
description: Use when writing MQL (Mondoo Query Language) queries, generating MQL from a check's title and description, or developing security policies
---

# MQL Development Skill

## Overview

This skill provides guidance for writing MQL (Mondoo Query Language) queries, generating them from natural-language check descriptions, and validating them using the cnspec CLI.

**Two-tier knowledge system:**
- **Reference Files** (static): MQL syntax docs, platform-specific examples, correctness traps
- **Schema Tools** (live): Real-time schema lookup and query validation via the cnspec CLI

## When to Use

- Writing MQL queries or policies
- Validating MQL syntax before deployment
- Exploring available MQL resources and fields
- Platform-specific query development (AWS, Azure, Linux, Windows, Microsoft 365)

## Reference Materials

Located within this skill directory:

| File | Purpose |
|------|---------|
| [mql-reference.md](mql-reference.md) | Complete MQL syntax and patterns |
| [samples/general.md](samples/general.md) | General MQL patterns |
| [samples/aws.md](samples/aws.md) | AWS resource patterns |
| [samples/azure.md](samples/azure.md) | Azure resource patterns |
| [samples/linux.md](samples/linux.md) | Linux system patterns |
| [samples/windows.md](samples/windows.md) | Windows system patterns |
| [samples/ms365.md](samples/ms365.md) | Microsoft 365 patterns |

## Schema Discovery & Query Validation

The cnspec CLI is the single source of schema data and query validation. It provides structured JSON output for all schema operations and works in any environment where cnspec is installed.

#### List all providers

```bash
cnspec providers list --json
```

Returns an array of providers with name, version, and connectors:
```json
[
  {"name": "aws", "version": "13.6.2", "connectors": ["aws"]},
  {"name": "os", "version": "13.8.1", "connectors": ["local", "ssh", "docker"]}
]
```

#### Get provider details (connectors and flags)

```bash
cnspec providers info aws --json
cnspec providers info aws azure --json   # multiple providers
```

Returns connector details including available flags for each connection type.

#### List resources in a provider

```bash
cnspec providers resources aws --json
```

Returns all resources with name, title, and field count:
```json
{
  "provider": "aws",
  "total_resources": 111,
  "resources": [
    {"name": "aws.ec2.instance", "title": "Amazon EC2 Instance", "field_count": 52}
  ]
}
```

#### Get resource field details

```bash
cnspec providers resources aws aws.ec2.instance --json
```

Returns all fields with types and descriptions:
```json
{
  "name": "aws.ec2.instance",
  "title": "Amazon EC2 Instance",
  "fields": [
    {"name": "arn", "type": "string", "title": "Amazon Resource Name"},
    {"name": "tags", "type": "map[string]string", "title": "Instance tags"}
  ]
}
```

#### Validate MQL queries

```bash
# Full compilation check — fails with exit 1 on invalid resources/fields
cnspec run local -c "asset.name" --ast

# Lexical parse only — checks syntax, NOT resource/field validity
cnspec run local -c "asset.name" --parse
```

**Important**: `--parse` accepts syntactically valid but semantically wrong queries (e.g., `invalid.bogus.thing` parses with exit 0). Use `--ast` to catch invalid resource or field names.

#### Execute queries

```bash
cnspec run local -c "users { name uid }" --json
```

#### Policy management

```bash
# Lint a policy bundle with structured SARIF output
cnspec policy lint policy.mql.yaml -o sarif

# Format a policy bundle to standard style (modifies file in place)
cnspec policy format policy.mql.yaml

# Sort and format a policy bundle
cnspec policy format policy.mql.yaml --sort

# Generate an example policy bundle scaffold
cnspec policy init example.mql.yaml
```

### Find similar existing checks (grounding)

Before writing MQL from scratch, look for a validated check that already does
something similar and mirror its patterns — real MQL for the same provider is
the strongest guide you have.

```bash
cnspec policy graph search "s3 buckets must be encrypted" ./content --similar --provider aws
cnspec policy graph search "containers must not run as root" ./content --similar --limit 5 --json
```

### When to Use What

| Need | Best Option |
|------|-------------|
| MQL syntax patterns | `mql-reference.md` |
| Platform-specific examples | `samples/*.md` |
| Resource availability check | `cnspec providers resources <provider> --json` |
| Field types and descriptions | `cnspec providers resources <provider> <resource> --json` |
| Query compilation validation | `cnspec run local -c "query" --ast` |
| Policy structure validation | `cnspec policy lint file.mql.yaml -o sarif` |

## MQL Quick Reference

### Core Syntax

```mql
# Basic resource access
resource.property == value

# Filtering
resources.where(condition).all(assertion)

# Data blocks
resource {
  property1
  property2 == expected_value
}

# Variables
v = 23
value = null

# Regular expression matching (NOT =~)
string == /pattern/

# Empty checks
value == empty
value != empty
```

### List Operations

```mql
# All entries must match
array.all(condition)

# At least one entry matches
array.contains(condition)

# No entries match
array.none(condition)

# Exactly one entry matches
array.one(condition)

# Filter entries
array.where(condition)

# Current item reference
array.where(_.contains("pattern"))
```

### Common Patterns

```mql
# File permissions
file("/etc/passwd").permissions {
  user_readable == true
  user_writeable == true
  group_readable == true
  other_readable == true
}

# Service status
service("ssh").running == true
service("telnet").enabled == false

# Package check
package("nginx").installed == true

# Kernel parameters
kernel.parameters['net.ipv4.ip_forward'] == 0

# Platform detection
asset.platform == "ubuntu"
asset.family.contains("linux")
```

### Anti-Patterns to Avoid

```mql
# Don't use =~ for regex
string =~ /pattern/      # Bad
string == /pattern/      # Good

# Don't use deprecated platform
platform == "ubuntu"          # Bad
asset.platform == "ubuntu"    # Good

# Don't nest .where() clauses
events.where(parameters.where(_['name'] == "NEW_VALUE"))  # Bad
events.where(parameters.any(_['name'] == "NEW_VALUE"))    # Good

# Always handle null values
users.all(shell == "/bin/bash")                     # Bad
users.where(shell != null).all(shell == "/bin/bash") # Good
```

### Correctness Traps (a query can compile and still be wrong)

Compilation (`--ast`) catches unknown resources, unknown fields, and syntax
errors. It does NOT catch these semantic traps, which cause a check to return a
confidently wrong verdict. Follow these rules — they are the difference between
MQL that compiles and MQL that is correct:

- **Assert presence before comparing.** `null && null` evaluates to **`true`**,
  so a check that only compares fields silently *passes* when the data never
  resolved. Write `field != empty && field == "x"`, not `field == "x"` alone.
- **Use `!= empty`, never `!= ""`** for non-empty checks — `null != ""` is true,
  so `!= ""` is not null-safe.
- **`.all()` / `.none()` on `null` errors**; on an empty list they pass
  vacuously. Guard absent collections: `x == empty || x.all(...)`.
- **A dotted path that is also a resource name compiles to an empty husk** — every
  field then reads `null` and `null != "x"` is true. Reach sub-objects through an
  accessor path or a block bound to the parent: `parent { child.field != "none" }`.
- **No parenthesized grouping** of booleans. Rely on precedence (`&&` binds
  tighter than `||`) or split into separate `.any()` / `.all()` calls.
- **Newline-as-AND**: multiple lines in an `mql:` block are implicitly AND-ed.
- **`filters:` is asset selection, not logic.** Keep `asset.platform == ...` in
  `filters:`; predicate logic belongs in `mql:`.

## Generating MQL from a check's description

`cnspec policy generate` fills in `mql:` for checks that have a `title` and
`docs.desc` but no query yet. cnspec resolves each check's target provider from
its `filters:`, searches similar existing checks for grounding, and validates
the generated MQL by compiling it — delegating the model call to a coding-agent
CLI you already have installed (Claude Code, Codex, Kimi, DeepSeek).

```bash
cnspec policy generate --interactive                   # guided: describe → generate → review → write
cnspec policy generate policy.mql.yaml --in-place      # fill empty mql, write back
cnspec policy generate policy.mql.yaml --dry-run       # preview without writing
cnspec policy generate policy.mql.yaml --force         # also regenerate existing mql
cnspec policy generate policy.mql.yaml --agent codex --explain
cnspec policy generate policy.mql.yaml --test-target aws  # execute-and-assert, not just compile
```

`--interactive` (`-i`) is the guided authoring flow: it asks what the check
should verify, guesses the provider and filter, shows similar existing checks as
grounding, generates the MQL, and lets you accept / edit / regenerate-with-
feedback before writing it into a bundle — one check at a time.

By default generated MQL is validated by compiling it. Pass `--test-target
<provider>` (e.g. `local`, `aws`, `gcp`, `azure`) to additionally run each query
against a live asset, or `--test-recording <file>` to run against a recording
(reproducible, no live credentials), and require it to resolve to a concrete
true/false verdict. This catches the correctness traps below — a query that
compiles but resolves to `null` (null-unsafe access, unresolved field, or a
dotted-path husk) is rejected.

When generating MQL by hand for a check, follow the same loop cnspec does:
1. Read the intent from `title` + `docs.desc`.
2. Resolve the provider from `filters:` (`asset.platform == "..."`).
3. `cnspec policy graph search "<intent>" <path> --similar --provider <p>` to find checks to mirror.
4. Confirm fields with `cnspec providers resources <provider> <resource> --json`.
5. Apply the correctness traps above.
6. Validate: `cnspec run <connection> -c "<mql>" --ast`.

## Workflow

1. **Understand requirements** - What resources need to be checked?
2. **Explore schema** - Use `cnspec providers resources <provider> --json`
3. **Find similar checks** - `cnspec policy graph search "<intent>" <path> --similar --provider <p>`
4. **Check samples** - Look for similar patterns in `samples/*.md`
5. **Write query** - Follow patterns from `mql-reference.md` and the correctness traps
6. **Validate** - Use `cnspec run local -c "query" --ast` to verify it compiles
7. **Test** - Run with `cnspec run` against target systems

## Platform-Specific Guidance

### AWS
- Use `aws.*` resources
- Check `samples/aws.md` for IAM, EC2, S3 patterns
- Explore: `cnspec providers resources aws --json`

### Azure
- Use `azure.subscription.*` resources
- Check `samples/azure.md` for VM, storage, security patterns
- Both full subscription and single resource scan patterns

### Linux
- Use `file`, `service`, `package`, `users`, `kernel` resources
- Check `samples/linux.md` for common patterns
- Handle platform variants (debian, redhat, etc.)

### Windows
- Use `registrykey`, `secpol`, `auditpol`, `windows` resources
- Check `samples/windows.md` for registry and policy patterns
- Handle server vs workstation differences

### Microsoft 365
- Use `microsoft.*` resources
- Check `samples/ms365.md` for domain patterns
