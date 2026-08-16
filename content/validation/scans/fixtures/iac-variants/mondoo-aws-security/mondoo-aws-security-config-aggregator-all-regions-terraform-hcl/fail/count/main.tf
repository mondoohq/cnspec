# Non-compliant: a counted aggregator does not cover all regions.
resource "aws_config_configuration_aggregator" "counted" {
  count = 2
  name  = "example-${count.index}"
  account_aggregation_source {
    account_ids = ["123456789012"]
    regions     = ["us-east-1"]
    all_regions = false
  }
}
