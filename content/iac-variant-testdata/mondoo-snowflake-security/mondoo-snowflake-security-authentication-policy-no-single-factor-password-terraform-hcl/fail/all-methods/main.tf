# ALL includes PASSWORD, so the policy permits single-factor password logins.
resource "snowflake_authentication_policy" "permissive" {
  database = snowflake_database.security.name
  schema   = snowflake_schema.policies.name
  name     = "PERMISSIVE"

  authentication_methods = ["ALL"]
  mfa_enrollment         = "OPTIONAL"
}
