resource "snowflake_password_policy" "standard" {
  database          = "SECURITY"
  schema            = "POLICIES"
  name              = "STANDARD"
  max_retries       = 5
  lockout_time_mins = 15
}
