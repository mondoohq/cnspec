# Non-compliant: one of two workgroups does not enforce its configuration.
resource "aws_athena_workgroup" "ok" {
  name = "pass-example"

  configuration {
    enforce_workgroup_configuration = true
  }
}

resource "aws_athena_workgroup" "bad" {
  name = "fail-example"

  configuration {
    enforce_workgroup_configuration = false
  }
}
