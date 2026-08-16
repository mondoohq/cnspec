# Known bug: provider: jsonencode list

The mql terraform provider returns an empty value for `jsonencode([list])`, so a check reading the encoded structure cannot evaluate this fixture correctly. Tracked as a provider fix.

Remove this marker when the underlying fix lands and this scenario asserts correctly.
