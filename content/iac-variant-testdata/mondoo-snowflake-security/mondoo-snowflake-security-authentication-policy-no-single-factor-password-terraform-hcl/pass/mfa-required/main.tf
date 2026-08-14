resource "snowflake_authentication_policy" "humans" {
  database = snowflake_database.security.name
  schema   = snowflake_schema.policies.name
  name     = "HUMAN_USERS"

  authentication_methods = ["PASSWORD", "SAML"]
  mfa_enrollment         = "REQUIRED"
}
