# Non-compliant: one of two aggregators does not cover all regions.
resource "aws_config_configuration_aggregator" "ok" {
  name = "ok"
  account_aggregation_source {
    account_ids = ["123456789012"]
    all_regions = true
  }
}

resource "aws_config_configuration_aggregator" "bad" {
  name = "bad"
  account_aggregation_source {
    account_ids = ["123456789012"]
    regions     = ["us-east-1"]
    all_regions = false
  }
}
