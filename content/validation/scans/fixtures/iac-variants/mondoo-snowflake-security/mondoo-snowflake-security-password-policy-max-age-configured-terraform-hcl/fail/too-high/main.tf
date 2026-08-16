resource "snowflake_password_policy" "loose" {
  database     = "SECURITY"
  schema       = "POLICIES"
  name         = "LOOSE"
  max_age_days = 120
}
