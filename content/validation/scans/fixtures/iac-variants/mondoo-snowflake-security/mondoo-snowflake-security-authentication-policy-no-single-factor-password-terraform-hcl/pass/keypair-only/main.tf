# Password is not an accepted method at all, so there is no single-factor path
# even though MFA enrollment is left optional.
resource "snowflake_authentication_policy" "service_accounts" {
  database = snowflake_database.security.name
  schema   = snowflake_schema.policies.name
  name     = "SERVICE_ACCOUNTS"

  authentication_methods = ["KEYPAIR", "OAUTH"]
  mfa_enrollment         = "OPTIONAL"
}
