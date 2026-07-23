# Non-compliant: config recorder status is disabled.
resource "aws_config_configuration_recorder_status" "fail_example" {
  name       = "fail-example"
  is_enabled = false
}
