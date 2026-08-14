resource "snowflake_authentication_policy" "humans" {
  database = snowflake_database.security.name
  schema   = snowflake_schema.policies.name
  name     = "HUMAN_USERS"

  authentication_methods = ["SAML", "PASSWORD"]
  mfa_authentication_methods = ["PASSWORD"]
  mfa_enrollment         = "REQUIRED"
  client_types           = ["SNOWFLAKE_UI", "DRIVERS"]
  comment                = "Interactive users must enroll in MFA"
}
