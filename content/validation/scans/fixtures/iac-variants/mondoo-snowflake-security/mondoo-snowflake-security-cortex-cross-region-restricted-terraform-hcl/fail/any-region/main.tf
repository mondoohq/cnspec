# ANY_REGION lets Snowflake process Cortex requests in any region on any cloud.
resource "snowflake_account_parameter" "cortex_enabled_cross_region" {
  key   = "CORTEX_ENABLED_CROSS_REGION"
  value = "any_region"
}
