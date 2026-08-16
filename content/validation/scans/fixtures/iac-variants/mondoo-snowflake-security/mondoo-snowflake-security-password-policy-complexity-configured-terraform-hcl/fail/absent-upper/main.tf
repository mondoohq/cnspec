resource "snowflake_password_policy" "partial" {
  database             = "SECURITY"
  schema               = "POLICIES"
  name                 = "PARTIAL"
  min_lower_case_chars = 1
  min_numeric_chars    = 1
  min_special_chars    = 1
}
