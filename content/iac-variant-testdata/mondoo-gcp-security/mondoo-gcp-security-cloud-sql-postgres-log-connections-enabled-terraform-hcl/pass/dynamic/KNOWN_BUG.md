# Known bug: provider: dynamic blocks

The mql terraform provider parses a `dynamic "x" { content {...} }` block as type `dynamic` with the real content nested under `content`, so `blocks.where(type == "x")` does not see it. Checks that iterate nested blocks this way cannot evaluate the dynamic form correctly until the provider normalizes `dynamic "x"` into a type-`x` block. Tracked as a provider fix.

Remove this marker when the underlying fix lands and this scenario asserts correctly.

## Re-verified 2026-08-14 — still broken, upstream issue closed

Re-run against terraform provider **v13.3.13**, downloaded into a clean
providers directory the way CI does, so this is not a stale-local-install
result. This scenario still asserts incorrectly.

mondoohq/mql#9077 is closed as *completed*, but the fix does not cover this case: all
21 dynamic-block scenarios in this corpus still fail against v13.3.13, and none
of them flipped. **Do not delete this marker on the strength of the issue being
closed** — the marker is load-bearing and the harness will tell you if the
scenario ever starts asserting correctly.
