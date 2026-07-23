# Compliant: organization aggregation source covers all regions.
resource "aws_config_configuration_aggregator" "pass_org" {
  name = "pass-org"

  organization_aggregation_source {
    all_regions = true
    role_arn    = "arn:aws:iam::123456789012:role/config"
  }
}
