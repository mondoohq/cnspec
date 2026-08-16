resource "snowflake_password_policy" "noretry" {
  database    = "SECURITY"
  schema      = "POLICIES"
  name        = "NO_RETRY"
  max_retries = 0
}
