---
name: mql
description: Use when writing MQL (Mondoo Query Language) queries, working with Mondoo MCP tools, or developing security policies
---

# MQL Development Skill

## Overview

This skill provides guidance for writing MQL (Mondoo Query Language) queries and validating them using either the cnspec CLI or Mondoo's MCP tools.

**Two-tier knowledge system:**
- **Reference Files** (static): MQL syntax docs, platform-specific examples
- **Schema Tools** (live): Real-time schema lookup and query validation via cnspec CLI or MCP

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

Two equivalent interfaces are available for real-time schema lookup and query validation. Use whichever is available in your environment — they provide the same data.

### cnspec CLI (recommended — works everywhere)

The cnspec CLI provides structured JSON output for all schema operations. No MCP server required.

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

### Mondoo MCP Server Tools (alternative)

If the Mondoo MCP server is available, you can use these tools instead of the CLI.

| MCP Tool | CLI Equivalent |
|----------|---------------|
| `mcp__mondoo-mcp-http__mql-schema-providers` | `cnspec providers list --json` |
| `mcp__mondoo-mcp-http__mql-schema-overview` | `cnspec providers resources <provider> --json` |
| `mcp__mondoo-mcp-http__mql-schema-resource` | `cnspec providers resources <provider> <resource> --json` |
| `mcp__mondoo-mcp-http__mql-schema-suggestion` | No CLI equivalent (use LSP) |
| `mcp__mondoo-mcp-http__mql-compiler` | `cnspec run local -c "query" --ast` |
| `mcp__mondoo-mcp-http__mql-bundle-lint` | `cnspec policy lint file.mql.yaml -o sarif` |
| `mcp__mondoo-mcp-http__mql-bundle-format` | `cnspec policy format file.mql.yaml` |
| `mcp__mondoo-mcp-http__mql-policy-bundle` | `cnspec policy init file.mql.yaml` |

### When to Use What

| Need | Best Option |
|------|-------------|
| MQL syntax patterns | `mql-reference.md` |
| Platform-specific examples | `samples/*.md` |
| Resource availability check | `cnspec providers resources <provider> --json` |
| Field types and descriptions | `cnspec providers resources <provider> <resource> --json` |
| Query compilation validation | `cnspec run local -c "query" --ast` |
| Policy structure validation | `cnspec policy lint file.mql.yaml -o sarif` |

## Wiring Policies into Content Validation

`cnspec policy lint` proves a policy **compiles**. It does not prove a check reaches the verdict you claim, that an IaC variant ever matches an asset, or that a remediation snippet is well-formed and actually fixes the thing it documents. Those live in a separate suite.

When you are authoring or editing policies **inside the cnspec repository**, every one of those checks lives in `content/validation/`, and [`content/validation/README.md`](https://github.com/mondoohq/cnspec/blob/main/content/validation/README.md) is the reference for all of it. Authoring rules are in [`content/CLAUDE.md`](https://github.com/mondoohq/cnspec/blob/main/content/CLAUDE.md).

```bash
make test/content              # lint + bundle scans + compliance mappings
make test/content/lint         # run this first, and fix it first
make test/content/iac          # IaC variant fixture suites
make test/content/iac/coverage # every IaC variant has pass+fail fixtures
make test/content/remediation  # remediation code-block linters
make test/content/commands     # remediation CLI and API validators
```

**A new policy has to be registered, or nothing validates it.** Every validator except the shell one and the compliance suites is allowlist-driven: it iterates a `TARGETS` dict or a policy slice. A bundle absent from those lists is not reported as unvalidated, it is never visited, so it merges with its variants untested and its remediation unlinted while CI stays green. Register the policy in the same change that adds it:

| Add the policy to | When |
|---|---|
| `content/README.md` | always — the user-facing catalog |
| `tfVariantPolicies` in `content/validation/scans/iac_variants_test.go` | it has `-terraform-hcl` / `-terraform-plan` / `-terraform-state` / `-cloudformation` / `-bicep` variants |
| `TARGETS` in `content/validation/remediation/code/<language>.py` | it ships that language's remediation. Terraform also needs a `PROVIDER_MAP` entry per resource prefix |
| a registry under `content/validation/remediation/commands/` | it ships `id: cli` / `id: api` blocks, or `audit:` steps that invoke a CLI or REST call |

**Every IaC variant needs pass and fail fixtures** under `content/validation/scans/fixtures/iac-variants/<policy>/<variant-uid>/{pass,fail}/<scenario>/`, and a coverage gate enforces it at 100%. A variant that asserts exactly what its own `filters:` require has no possible failing input; record that with a `fail/IMPOSSIBLE.md` explaining why, which is the only sanctioned way to ship without a fail fixture.

Scope a run to one check rather than running the whole suite — the trailing slash is what makes it work:

```bash
go test -tags iac_variants ./content/validation/scans \
  -run 'TestTerraformVariants/mondoo-aws-security/mondoo-aws-security-s3-bucket-encryption-terraform-hcl/'
```

Three outcomes are distinguished, and the third matters most: **passed**, **failed**, and **skipped** — the check never ran because no asset matched its `filters:`. A skipped check is a fixture bug, not a pass. It looks identical to a passing one in every report, forever.

**A policy with IaC variants leaves its groups unfiltered.** A group filter is evaluated before the check's own filter, so a group carrying `filters: asset.platform == "<api-platform>"` means a Terraform asset never reaches the variant underneath it and every fixture reports as *skipped*. Let each variant's own `filters:` select its asset instead, and convert the groups in the same change that adds the first variant.

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

## Workflow

1. **Understand requirements** - What resources need to be checked?
2. **Explore schema** - Use `cnspec providers resources <provider> --json`
3. **Check samples** - Look for similar patterns in `samples/*.md`
4. **Write query** - Follow patterns from `mql-reference.md`
5. **Validate** - Use `cnspec run local -c "query" --ast` to verify syntax
6. **Test** - Run with `cnspec run` against target systems
7. **Wire up validation** - For policies in the cnspec repository, add fixtures for any IaC variant and register the policy with the validators that cover its remediation, in the same change. See [Wiring Policies into Content Validation](#wiring-policies-into-content-validation)

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
