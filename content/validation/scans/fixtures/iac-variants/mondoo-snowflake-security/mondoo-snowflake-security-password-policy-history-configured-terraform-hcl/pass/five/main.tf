resource "snowflake_password_policy" "standard" {
  database = "SECURITY"
  schema   = "POLICIES"
  name     = "STANDARD"
  history  = 5
}
