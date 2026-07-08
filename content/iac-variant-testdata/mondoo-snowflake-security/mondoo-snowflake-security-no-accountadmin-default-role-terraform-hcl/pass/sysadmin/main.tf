resource "snowflake_user" "analyst" {
  name         = "ANALYST"
  default_role = "SYSADMIN"
}
