# mql

An agent skill for MQL (Mondoo Query Language) development with syntax guidance, platform-specific patterns, and MCP tool integration.

## What it does

Provides comprehensive guidance for writing MQL queries and security policies:

- **MQL Reference** - Complete syntax documentation, best practices, and anti-patterns to avoid
- **Platform Samples** - Ready-to-use patterns for AWS, Azure, Linux, Windows, and Microsoft 365
- **Schema Discovery** - Real-time schema lookup via cnspec CLI or Mondoo MCP tools
- **Query Validation** - Compile-time syntax and semantic checking
- **Policy Management** - Linting, formatting, and scaffolding policy bundles

## Usage

The skill automatically activates when working on MQL-related tasks. You can also invoke it directly:

```
/mql
```

## Installation

See [skills/README.md](../README.md) for installation instructions across Claude Code, Codex, Gemini CLI, and Cursor.
