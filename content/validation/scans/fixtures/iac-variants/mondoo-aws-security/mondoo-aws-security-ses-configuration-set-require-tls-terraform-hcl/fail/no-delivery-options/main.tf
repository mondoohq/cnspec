# With no delivery_options the set inherits the OPTIONAL default.
resource "aws_sesv2_configuration_set" "transactional" {
  configuration_set_name = "transactional"

  reputation_options {
    reputation_metrics_enabled = true
  }
}
