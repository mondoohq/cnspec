resource "snowflake_user" "legacy_admin" {
  name         = "LEGACY_ADMIN"
  default_role = "ACCOUNTADMIN"
  disabled     = true
}
