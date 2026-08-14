# Known bug: provider: jsonencode list

The mql terraform provider returns an empty value for `jsonencode([list])`, so a check reading the encoded structure cannot evaluate this fixture correctly. Tracked as a provider fix.

Remove this marker when the underlying fix lands and this scenario asserts correctly.

## Re-verified 2026-08-14 — still broken, upstream issue closed

Re-run against terraform provider **v13.3.13**, downloaded into a clean
providers directory the way CI does, so this is not a stale-local-install
result. This scenario still asserts incorrectly.

mondoohq/mql#9079 is closed as *completed*, but the fix does not cover this case: all
2 jsonencode-list scenarios in this corpus still fail against v13.3.13, and none
of them flipped. **Do not delete this marker on the strength of the issue being
closed** — the marker is load-bearing and the harness will tell you if the
scenario ever starts asserting correctly.
