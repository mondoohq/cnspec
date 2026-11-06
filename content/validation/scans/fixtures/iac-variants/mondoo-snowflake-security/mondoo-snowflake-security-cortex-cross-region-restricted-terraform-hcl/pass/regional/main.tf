# Regional values keep inference within one cloud geography.
resource "snowflake_account_parameter" "cortex_enabled_cross_region" {
  key   = "CORTEX_ENABLED_CROSS_REGION"
  value = "AWS_EU,AZURE_EU"
}

resource "snowflake_account_parameter" "statement_timeout" {
  key   = "STATEMENT_TIMEOUT_IN_SECONDS"
  value = "3600"
}
