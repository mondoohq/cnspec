# Compliant: recorder records all supported resources and global resource types.
resource "aws_config_configuration_recorder" "pass_example" {
  name     = "pass-example"
  role_arn = "arn:aws:iam::123456789012:role/config"

  recording_group {
    all_supported                 = true
    include_global_resource_types = true
  }
}
