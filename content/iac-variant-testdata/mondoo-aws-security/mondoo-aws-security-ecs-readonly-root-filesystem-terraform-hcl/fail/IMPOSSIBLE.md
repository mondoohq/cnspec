# No possible failing fixture on idiomatic Terraform HCL

`container_definitions` is a JSON **string** argument (`jsonencode([ ... ])`). The
provider evaluates a `jsonencode([...])` array to an **empty list** `[]`, so
`arguments.container_definitions.all(_['readonlyRootFilesystem'] == true)` is
vacuously true — a container with `readonlyRootFilesystem = false` still passes.
The check cannot flag any misconfiguration. Reported as a provider/mql bug.
