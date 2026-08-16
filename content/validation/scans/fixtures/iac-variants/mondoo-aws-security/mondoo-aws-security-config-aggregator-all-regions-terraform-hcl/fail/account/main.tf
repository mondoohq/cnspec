# Non-compliant: account aggregation source does not cover all regions.
resource "aws_config_configuration_aggregator" "fail_account" {
  name = "fail-account"

  account_aggregation_source {
    account_ids = ["123456789012"]
    regions     = ["us-east-1"]
    all_regions = false
  }
}
