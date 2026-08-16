# Known bug: policy: check does not yet assert this form

The `mondoo-aws-security-glue-crawler-security-configuration-cloudformation` check does not yet assert this realistic fixture correctly. This is a harness-found policy or provider issue whose fix is tracked outside the test framework pull request; the fixture is kept as a regression test.

Remove this marker when the underlying fix lands and this scenario asserts correctly.
