resource "snowflake_account_parameter" "cortex_enabled_cross_region" {
  key   = "CORTEX_ENABLED_CROSS_REGION"
  value = "DISABLED"
}
