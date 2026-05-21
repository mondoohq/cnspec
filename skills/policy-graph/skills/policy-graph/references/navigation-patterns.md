# Navigation Patterns

Common patterns for using cnspec policy graph to answer questions about policy bundles.

## "What checks does this policy have?"

```bash
cnspec policy graph callees mondoo-linux-security ./content/mondoo-linux-security.mql.yaml
```

Shows all groups, then use callees on a group to see its checks.

## "Which compliance controls map to this check?"

```bash
cnspec policy graph callers <check-uid> ./content/
```

Look for `[maps_to]` edges — those come from framework controls via framework_maps.

## "How does framework X relate to check Y?"

```bash
cnspec policy graph paths <framework-uid> <check-uid> ./content/
```

Shows the chain: framework → (contains) → control → (maps_to) → check.

## "Show me everything about this check"

```bash
cnspec policy graph context <check-uid> ./content/ --depth 2
```

Shows the check's MQL code, impact, compliance tags, which policy groups reference it, and which framework controls map to it — all with YAML source snippets.

## "What's the full structure of this policy?"

```bash
# First see the groups
cnspec policy graph callees <policy-uid> ./content/

# Then drill into a specific group
cnspec policy graph callees <group-id> ./content/

# Or get the full reachable set
cnspec policy graph reachable <policy-uid> ./content/
```

## "Find all SSH-related checks"

```bash
cnspec policy graph search ssh ./content/ --kind check
```

## "Find checks by title"

```bash
cnspec policy graph search "root login" ./content/ --kind check
```

## "What policies are in this bundle directory?"

```bash
cnspec policy graph search "" ./content/ --kind policy
```

## "Generate a visual diagram"

```bash
cnspec policy graph export ./content/mondoo-linux-security.mql.yaml --format dot > policy.dot
dot -Tpng policy.dot -o policy.png
```

## "Compare two policies"

```bash
# Export each and compare node lists
cnspec policy graph export ./content/mondoo-linux-security.mql.yaml --format json > linux.json
cnspec policy graph export ./content/mondoo-aws-security.mql.yaml --format json > aws.json
python3 -c "
import json
linux = {n['name'] for n in json.load(open('linux.json'))['nodes'] if n['kind'] == 'check'}
aws = {n['name'] for n in json.load(open('aws.json'))['nodes'] if n['kind'] == 'check'}
print(f'Linux-only checks: {len(linux - aws)}')
print(f'AWS-only checks: {len(aws - linux)}')
print(f'Shared checks: {len(linux & aws)}')
"
```
