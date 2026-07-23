resource "snowflake_account_parameter" "period" {
  key   = "DATA_RETENTION_TIME_IN_DAYS"
  value = "30"
}
