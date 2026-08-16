resource "aws_sesv2_configuration_set" "transactional" {
  configuration_set_name = "transactional"

  delivery_options {
    tls_policy = "REQUIRE"
  }

  reputation_options {
    reputation_metrics_enabled = true
  }
}
