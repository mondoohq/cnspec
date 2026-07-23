resource "snowflake_password_policy" "noage" {
  database   = "SECURITY"
  schema     = "POLICIES"
  name       = "NO_AGE"
  min_length = 14
}
