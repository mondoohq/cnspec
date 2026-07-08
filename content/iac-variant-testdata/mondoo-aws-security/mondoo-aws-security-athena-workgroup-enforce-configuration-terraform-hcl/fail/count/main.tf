# Non-compliant: a counted workgroup does not enforce its configuration.
resource "aws_athena_workgroup" "fail_count" {
  count = 2
  name  = "fail-example-${count.index}"

  configuration {
    enforce_workgroup_configuration = false
  }
}
