# Non-compliant: account aggregation source lists specific regions and omits
# all_regions, so it does not aggregate across all regions.
resource "aws_config_configuration_aggregator" "fail_account_no_flag" {
  name = "fail-account-no-flag"

  account_aggregation_source {
    account_ids = ["123456789012"]
    regions     = ["us-east-1", "us-west-2"]
  }
}
