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
