# AWS_GLOBAL allows any AWS region worldwide, crossing geographic boundaries.
resource "snowflake_account_parameter" "cortex_enabled_cross_region" {
  key   = "CORTEX_ENABLED_CROSS_REGION"
  value = "AWS_GLOBAL"
}
