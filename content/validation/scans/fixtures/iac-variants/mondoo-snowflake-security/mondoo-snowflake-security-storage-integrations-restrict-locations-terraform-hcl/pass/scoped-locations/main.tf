resource "snowflake_storage_integration" "analytics" {
  name    = "ANALYTICS_S3"
  type    = "EXTERNAL_STAGE"
  enabled = true

  storage_provider         = "S3"
  storage_aws_role_arn     = "arn:aws:iam::123456789012:role/snowflake-analytics"
  storage_allowed_locations = [
    "s3://example-analytics-raw/events/",
    "s3://example-analytics-curated/",
  ]
}
