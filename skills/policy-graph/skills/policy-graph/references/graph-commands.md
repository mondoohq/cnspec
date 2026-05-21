# cnspec policy graph Commands

## callers

Show all inbound edges to a node — what references it.

```bash
cnspec policy graph callers <uid> <path>
cnspec policy graph callers <uid> <path> --json
```

Returns for each inbound edge: the source node's qualified name, edge kind, and source location.

**Use cases**: find which policies include a check, which compliance controls map to a check, which frameworks reference a policy.

## callees

Show all outbound edges from a node — what it contains or references.

```bash
cnspec policy graph callees <uid> <path>
cnspec policy graph callees <uid> <path> --json
```

**Use cases**: list all checks in a policy, list all controls in a framework, see what a group contains.

## context

Generate LLM-friendly markdown with YAML source snippets for a node and its N-hop neighborhood.

```bash
cnspec policy graph context <uid> <path>
cnspec policy graph context <uid> <path> --depth 3
```

Output includes:
- Focus node metadata (kind, uid, title, location, impact, compliance tags)
- N-hop neighborhood grouped by file
- YAML source snippets in fenced code blocks
- Relationship labels: "Referenced by" (inbound), "Contains" (outbound)

Default depth is 2 hops.

## paths

Find all simple paths between two nodes.

```bash
cnspec policy graph paths <from-uid> <to-uid> <path>
cnspec policy graph paths <from-uid> <to-uid> <path> --json
```

Searches up to depth 20. Shows each path as a chain of qualified node names.

**Use cases**: trace how a framework control connects to a specific check through the framework_map → control → check chain.

## reachable

Show all nodes transitively reachable by following outbound edges.

```bash
cnspec policy graph reachable <uid> <path>
cnspec policy graph reachable <uid> <path> --json
```

**Use cases**: find all checks reachable from a policy, find all nodes in a framework's scope.

## export

Export the full policy graph.

```bash
cnspec policy graph export <path> --format json
cnspec policy graph export <path> --format dot
```

Formats:
- `json`: Full graph with nodes and edges arrays
- `dot`: Graphviz DOT format for visualization

**Use cases**: understand the full structure of a bundle, generate visual diagrams, programmatic analysis.

## Common Flags

- `--json`: Output as JSON (available on callers, callees, paths, reachable)
- `--depth N`: Neighborhood depth for context (default 2)
- `--format json|dot`: Export format (export command only)

## Path Argument

The `<path>` argument can be:
- A directory: walks for all `.mql.yaml` files recursively
- One or more specific `.mql.yaml` files
- Multiple paths: `cnspec policy graph callers check-uid file1.mql.yaml file2.mql.yaml`
