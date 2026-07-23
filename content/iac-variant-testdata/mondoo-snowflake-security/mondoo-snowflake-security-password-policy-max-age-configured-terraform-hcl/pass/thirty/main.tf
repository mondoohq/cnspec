resource "snowflake_password_policy" "strict" {
  database     = "SECURITY"
  schema       = "POLICIES"
  name         = "STRICT"
  max_age_days = 30
}
