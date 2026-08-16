# "*" allows a stage on any bucket the integration's IAM role can reach, which
# defeats the point of scoping the integration.
resource "snowflake_storage_integration" "analytics" {
  name    = "ANALYTICS_S3"
  type    = "EXTERNAL_STAGE"
  enabled = true

  storage_provider          = "S3"
  storage_aws_role_arn      = "arn:aws:iam::123456789012:role/snowflake-analytics"
  storage_allowed_locations = ["*"]
}
