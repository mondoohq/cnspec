# Non-compliant: a counted recorder does not record global resource types.
resource "aws_config_configuration_recorder" "counted" {
  count    = 2
  name     = "example-${count.index}"
  role_arn = "arn:aws:iam::123456789012:role/config"
  recording_group {
    all_supported                 = true
    include_global_resource_types = false
  }
}
