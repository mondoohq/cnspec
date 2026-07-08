# Compliant: recording group configured via a conditional dynamic block.
variable "record_all" {
  type    = bool
  default = true
}

resource "aws_config_configuration_recorder" "pass_dynamic" {
  name     = "pass-dynamic"
  role_arn = "arn:aws:iam::123456789012:role/config"

  dynamic "recording_group" {
    for_each = var.record_all ? [1] : []
    content {
      all_supported                 = true
      include_global_resource_types = true
    }
  }
}
