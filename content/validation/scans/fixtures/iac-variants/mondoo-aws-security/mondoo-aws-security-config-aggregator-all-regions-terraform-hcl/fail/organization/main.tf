# Non-compliant: organization aggregation source does not cover all regions.
resource "aws_config_configuration_aggregator" "fail_org" {
  name = "fail-org"

  organization_aggregation_source {
    all_regions = false
    regions     = ["us-east-1"]
    role_arn    = "arn:aws:iam::123456789012:role/config"
  }
}
