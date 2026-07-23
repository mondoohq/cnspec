resource "snowflake_password_policy" "standard" {
  database   = "SECURITY"
  schema     = "POLICIES"
  name       = "STANDARD"
  min_length = 14
}
