# Inline credentials are long-lived keys stored in the stage definition, and
# they end up in the Terraform state file as well.
resource "snowflake_stage" "raw_events" {
  name        = "RAW_EVENTS"
  database    = snowflake_database.analytics.name
  schema      = snowflake_schema.landing.name
  url         = "s3://example-analytics-raw/events/"
  credentials = "AWS_KEY_ID='AKIAIOSFODNN7EXAMPLE' AWS_SECRET_KEY='wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY'"
}
