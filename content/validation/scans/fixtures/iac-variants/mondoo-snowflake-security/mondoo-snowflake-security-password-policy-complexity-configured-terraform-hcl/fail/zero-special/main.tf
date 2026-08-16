resource "snowflake_password_policy" "weak" {
  database             = "SECURITY"
  schema               = "POLICIES"
  name                 = "WEAK"
  min_upper_case_chars = 1
  min_lower_case_chars = 1
  min_numeric_chars    = 1
  min_special_chars    = 0
}
