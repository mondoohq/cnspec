resource "snowflake_password_policy" "never" {
  database     = "SECURITY"
  schema       = "POLICIES"
  name         = "NEVER_EXPIRE"
  max_age_days = 0
}
