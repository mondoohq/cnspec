# Compliant: max_age_days is omitted, so Snowflake applies its default
# PASSWORD_MAX_AGE_DAYS of 90, which is inside the required 1 to 90 day window.
resource "snowflake_password_policy" "noage" {
  database   = "SECURITY"
  schema     = "POLICIES"
  name       = "NO_AGE"
  min_length = 14
}
