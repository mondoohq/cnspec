# No possible failing fixture on idiomatic Terraform HCL

`container_definitions` is a JSON **string** argument (`jsonencode([ ... ])`). The
provider evaluates a `jsonencode([...])` array to an **empty list** `[]`, so
`arguments.container_definitions.all(_['logConfiguration'] != empty)` is vacuously
true for every task definition — a container with no `logConfiguration` still passes.
The check cannot flag any misconfiguration. Reported as a provider/mql bug.
