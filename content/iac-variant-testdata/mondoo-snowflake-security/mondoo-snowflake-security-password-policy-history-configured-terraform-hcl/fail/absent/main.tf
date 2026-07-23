resource "snowflake_password_policy" "nohistory" {
  database  = "SECURITY"
  schema    = "POLICIES"
  name      = "NO_HISTORY"
  min_length = 14
}
