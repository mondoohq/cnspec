# Known bug: provider: negative numeric literals

The mql terraform provider normalizes a negative numeric literal such as `-1` to a positive value, so a check comparing against a sentinel negative value cannot evaluate this fixture correctly. Tracked as a provider fix.

Remove this marker when the underlying fix lands and this scenario asserts correctly.
