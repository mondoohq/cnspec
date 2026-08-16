resource "snowflake_password_policy" "loose" {
  database    = "SECURITY"
  schema      = "POLICIES"
  name        = "LOOSE"
  max_retries = 15
}
