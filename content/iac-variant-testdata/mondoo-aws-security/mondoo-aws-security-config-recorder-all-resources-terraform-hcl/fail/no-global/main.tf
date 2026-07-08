# Non-compliant: recorder excludes global resource types.
resource "aws_config_configuration_recorder" "fail_example" {
  name     = "fail-example"
  role_arn = "arn:aws:iam::123456789012:role/config"

  recording_group {
    all_supported                 = true
    include_global_resource_types = false
  }
}
