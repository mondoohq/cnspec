# Compliant: a config configuration aggregator is defined.
resource "aws_config_configuration_aggregator" "pass_example" {
  name = "pass-example"

  account_aggregation_source {
    account_ids = ["123456789012"]
    all_regions = true
  }
}
