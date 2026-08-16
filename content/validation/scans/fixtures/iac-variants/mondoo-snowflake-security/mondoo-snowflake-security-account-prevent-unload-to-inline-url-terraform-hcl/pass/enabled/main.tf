resource "snowflake_account_parameter" "prevent_unload_to_inline_url" {
  key   = "PREVENT_UNLOAD_TO_INLINE_URL"
  value = "true"
}

resource "snowflake_account_parameter" "statement_timeout" {
  key   = "STATEMENT_TIMEOUT_IN_SECONDS"
  value = "3600"
}
