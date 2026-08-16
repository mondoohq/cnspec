# An enabled integration with no network rules attached does not constrain
# where a UDF or procedure may reach on the internet.
resource "snowflake_external_access_integration" "payments_api" {
  name                  = "PAYMENTS_API_ACCESS"
  allowed_network_rules = []
  enabled               = true
}
