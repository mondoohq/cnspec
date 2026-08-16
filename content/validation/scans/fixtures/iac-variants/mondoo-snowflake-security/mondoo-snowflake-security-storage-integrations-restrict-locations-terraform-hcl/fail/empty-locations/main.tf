# An enabled integration with no allowed locations places no constraint on
# where a stage built on it may point.
resource "snowflake_storage_integration" "analytics" {
  name    = "ANALYTICS_S3"
  type    = "EXTERNAL_STAGE"
  enabled = true

  storage_provider          = "S3"
  storage_aws_role_arn      = "arn:aws:iam::123456789012:role/snowflake-analytics"
  storage_allowed_locations = []
}
