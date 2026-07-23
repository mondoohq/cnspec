resource "snowflake_password_policy" "nolen" {
  database    = "SECURITY"
  schema      = "POLICIES"
  name        = "NO_LEN"
  max_retries = 5
}
