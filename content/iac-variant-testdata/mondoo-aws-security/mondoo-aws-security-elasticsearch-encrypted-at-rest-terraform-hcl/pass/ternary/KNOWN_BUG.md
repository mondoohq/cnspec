# Known bug: provider: unresolved conditionals

The mql terraform provider leaves `cond ? a : b` expressions unresolved during static analysis, so a scalar-equality assertion sees the unresolved value rather than the compliant literal. Tracked as a provider limitation.

Remove this marker when the underlying fix lands and this scenario asserts correctly.
