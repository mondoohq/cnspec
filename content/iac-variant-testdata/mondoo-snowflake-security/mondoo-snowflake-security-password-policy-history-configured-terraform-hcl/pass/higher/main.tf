resource "snowflake_password_policy" "strict" {
  database = "SECURITY"
  schema   = "POLICIES"
  name     = "STRICT"
  history  = 24
}
