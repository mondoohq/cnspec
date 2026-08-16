# Leaving this off lets any role create a stage with inline cloud credentials
# instead of going through a reviewed storage integration.
resource "snowflake_account_parameter" "require_storage_integration" {
  key   = "REQUIRE_STORAGE_INTEGRATION_FOR_STAGE_CREATION"
  value = "false"
}
