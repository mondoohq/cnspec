# No possible failing fixture: the filter asserts what the query asserts

The variant's `filters:` already requires an `aws_iam_role_policy_attachment`
whose `policy_arn` is `arn:aws:iam::aws:policy/AWSSupportAccess`:

```
asset.platform == "terraform-hcl" && terraform.resources.contains(
  nameLabel == "aws_iam_role_policy_attachment" &&
  arguments["policy_arn"] == "arn:aws:iam::aws:policy/AWSSupportAccess")
```

and the `mql:` then re-asserts the same condition:

```
terraform.resources.where(nameLabel == "aws_iam_role_policy_attachment").any(
  arguments['policy_arn'] == "arn:aws:iam::aws:policy/AWSSupportAccess")
```

Any configuration that satisfies the filter necessarily satisfies the query, and
any configuration that would fail the query is filtered out and never scored. The
variant can only ever be "passed" or "not applicable" — never "failed" — so no
failing input exists to write.

This is a real design gap, not a fixture gap: a check that cannot fail provides no
signal. It is tracked in issue #3118, which covers sweeping this class
(filter == MQL assertion) across the policy. Fixing it means broadening the filter
to the population that *should* have a support role (for example, any configuration
declaring `aws_iam_role`) so a config lacking the attachment is scored and fails.

When #3118 is addressed, delete this marker and add the real fail fixture.
