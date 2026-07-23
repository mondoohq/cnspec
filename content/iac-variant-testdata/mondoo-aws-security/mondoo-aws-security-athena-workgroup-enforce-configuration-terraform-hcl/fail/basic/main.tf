# Non-compliant: workgroup configuration does not enforce its settings.
resource "aws_athena_workgroup" "fail_example" {
  name = "fail-example"

  configuration {
    enforce_workgroup_configuration = false
  }
}
