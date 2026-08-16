# OPTIONAL enrollment means users may decline MFA entirely, so a stolen
# password is enough on its own.
resource "snowflake_authentication_policy" "humans" {
  database = snowflake_database.security.name
  schema   = snowflake_schema.policies.name
  name     = "HUMAN_USERS"

  authentication_methods = ["SAML", "PASSWORD"]
  mfa_enrollment         = "OPTIONAL"
}
