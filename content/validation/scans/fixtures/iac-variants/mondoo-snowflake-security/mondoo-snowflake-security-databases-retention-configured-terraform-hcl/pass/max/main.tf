resource "snowflake_database" "analytics" {
  name                        = "ANALYTICS"
  data_retention_time_in_days = 90
}
