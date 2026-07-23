resource "snowflake_password_policy" "noretry" {
  database   = "SECURITY"
  schema     = "POLICIES"
  name       = "NO_RETRY_SET"
  min_length = 14
}
