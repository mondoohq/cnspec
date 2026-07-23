resource "snowflake_user" "onboarding" {
  name                 = "TEMP_CONTRACTOR"
  must_change_password = true
  disabled             = true
}
