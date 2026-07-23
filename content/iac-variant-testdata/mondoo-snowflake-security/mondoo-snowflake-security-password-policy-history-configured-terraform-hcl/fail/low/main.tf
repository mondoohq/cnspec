resource "snowflake_password_policy" "weak" {
  database = "SECURITY"
  schema   = "POLICIES"
  name     = "WEAK"
  history  = 2
}
