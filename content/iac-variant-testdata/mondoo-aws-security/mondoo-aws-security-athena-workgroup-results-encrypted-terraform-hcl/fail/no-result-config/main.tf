# Non-compliant: configuration has no result_configuration block at all.
resource "aws_athena_workgroup" "fail_no_result_config" {
  name = "fail-no-result-config"

  configuration {
    enforce_workgroup_configuration = true
  }
}
