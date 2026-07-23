resource "snowflake_user" "admin" {
  name         = "PLATFORM_ADMIN"
  default_role = "ACCOUNTADMIN"
}
