# Known bug: provider: dynamic blocks

The mql terraform provider parses a `dynamic "x" { content {...} }` block as type `dynamic` with the real content nested under `content`, so `blocks.where(type == "x")` does not see it. Checks that iterate nested blocks this way cannot evaluate the dynamic form correctly until the provider normalizes `dynamic "x"` into a type-`x` block. Tracked as a provider fix.

Remove this marker when the underlying fix lands and this scenario asserts correctly.
