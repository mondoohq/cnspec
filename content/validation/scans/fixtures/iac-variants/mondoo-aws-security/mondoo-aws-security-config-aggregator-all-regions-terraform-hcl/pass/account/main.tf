# Compliant: account aggregation source covers all regions.
resource "aws_config_configuration_aggregator" "pass_account" {
  name = "pass-account"

  account_aggregation_source {
    account_ids = ["123456789012"]
    all_regions = true
  }
}
