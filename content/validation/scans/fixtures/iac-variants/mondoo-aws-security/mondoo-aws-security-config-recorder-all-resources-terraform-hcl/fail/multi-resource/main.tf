# Non-compliant: one of two recorders does not record all resources.
resource "aws_config_configuration_recorder" "ok" {
  name     = "ok"
  role_arn = "arn:aws:iam::123456789012:role/config"
  recording_group {
    all_supported                 = true
    include_global_resource_types = true
  }
}

resource "aws_config_configuration_recorder" "bad" {
  name     = "bad"
  role_arn = "arn:aws:iam::123456789012:role/config"
  recording_group {
    all_supported                 = false
    include_global_resource_types = false
  }
}
