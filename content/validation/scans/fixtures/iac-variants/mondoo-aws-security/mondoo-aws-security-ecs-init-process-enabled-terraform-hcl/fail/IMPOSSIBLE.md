# No possible failing fixture on idiomatic Terraform HCL

`container_definitions` is a JSON **string** argument, so the only valid/idiomatic
form is `jsonencode([ ... ])`. The Terraform provider evaluates a `jsonencode([...])`
(a top-level JSON array) to an **empty list** `[]`. The check
`arguments.container_definitions.all(_['linuxParameters']['initProcessEnabled'] == true)`
is therefore **vacuously true** for every task definition — a container with
`initProcessEnabled = false` still passes. The check cannot flag any misconfiguration.
See findings: this is a provider/mql bug, reported to the coordinator.
