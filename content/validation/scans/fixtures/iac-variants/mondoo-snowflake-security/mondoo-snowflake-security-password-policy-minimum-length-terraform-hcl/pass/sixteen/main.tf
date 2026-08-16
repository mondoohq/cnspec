resource "snowflake_password_policy" "strict" {
  database   = "SECURITY"
  schema     = "POLICIES"
  name       = "STRICT"
  min_length = 16
}
