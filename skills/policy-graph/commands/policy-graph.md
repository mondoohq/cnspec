---
name: policy-graph
description: Navigates cnspec policy/framework bundles using graph commands. Use when exploring policies, finding checks, tracing compliance mappings, or understanding policy structure.
argument-hint: "<question about policies, or specific navigation task>"
allowed-tools: Bash Read Glob Grep
---

# Policy Bundle Navigation with cnspec policy graph

**Task:** $ARGUMENTS

Use `cnspec policy graph` commands to answer the question or complete the investigation. The graph is built on the fly from `.mql.yaml` files — no setup needed.

## Workflow

Choose the appropriate phase based on the task:

### Phase 1: Orient (understand what's in the bundle)

```bash
# Export graph and count nodes by kind
cnspec policy graph export ./content/ --format json | python3 -c "
import json, sys, collections
d = json.load(sys.stdin)
kinds = collections.Counter(n['kind'] for n in d['nodes'])
print(f'Nodes: {len(d[\"nodes\"])}, Edges: {len(d[\"edges\"])}')
for k, v in kinds.most_common(): print(f'  {k}: {v}')
"
```

### Phase 2: Locate (find specific nodes)

```bash
# Find nodes by name, title, or UID
cnspec policy graph search ssh ./content/
cnspec policy graph search "root login" ./content/ --kind check
cnspec policy graph search "" ./content/ --kind policy
```

### Phase 3: Navigate (explore relationships)

```bash
# What references this check? (policies, groups, compliance controls)
cnspec policy graph callers <check-uid> ./content/

# What does this policy contain? (groups, checks, queries)
cnspec policy graph callees <policy-uid> ./content/

# Trace compliance mapping: framework control → check
cnspec policy graph paths <control-uid> <check-uid> ./content/

# All nodes reachable from a starting point
cnspec policy graph reachable <policy-uid> ./content/
```

### Phase 4: Context (deep dive with source)

```bash
# Full LLM-friendly context with YAML snippets
cnspec policy graph context <uid> ./content/ --depth 2
```

This shows the node, its neighbors, relationship labels, and the actual YAML source from the bundle files.

## Guidelines

- Use `--json` when you need to process results programmatically
- The `<path>` argument can be a directory or specific `.mql.yaml` files
- Node UIDs use substring matching — you don't need the full UID
- Use `context` for deep investigation — it shows source code and all relationships
- Use `callers` to understand "who uses this check" (compliance tracing)
- Use `callees` to understand "what's in this policy" (structure exploration)
