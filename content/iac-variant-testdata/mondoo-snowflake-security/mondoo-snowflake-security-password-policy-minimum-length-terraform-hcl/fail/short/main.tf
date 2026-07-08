resource "snowflake_password_policy" "weak" {
  database   = "SECURITY"
  schema     = "POLICIES"
  name       = "WEAK"
  min_length = 8
}
