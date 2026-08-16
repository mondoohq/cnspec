# Compliant: workgroup configuration enforces its settings.
resource "aws_athena_workgroup" "pass_example" {
  name = "pass-example"

  configuration {
    enforce_workgroup_configuration = true
  }
}
