resource "snowflake_user" "analyst" {
  name                 = "ANALYST"
  must_change_password = true
}
