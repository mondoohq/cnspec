# Compliant: config recorder status is enabled.
resource "aws_config_configuration_recorder_status" "pass_example" {
  name       = "pass-example"
  is_enabled = true
}
