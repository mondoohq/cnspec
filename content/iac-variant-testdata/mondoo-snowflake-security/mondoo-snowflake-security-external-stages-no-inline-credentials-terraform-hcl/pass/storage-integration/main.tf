resource "snowflake_stage" "raw_events" {
  name                = "RAW_EVENTS"
  database            = snowflake_database.analytics.name
  schema              = snowflake_schema.landing.name
  url                 = "s3://example-analytics-raw/events/"
  storage_integration = snowflake_storage_integration.analytics.name
  comment             = "Landing zone for raw event exports"
}
