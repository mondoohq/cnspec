resource "snowflake_account_parameter" "require_storage_integration" {
  key   = "REQUIRE_STORAGE_INTEGRATION_FOR_STAGE_CREATION"
  value = "true"
}
