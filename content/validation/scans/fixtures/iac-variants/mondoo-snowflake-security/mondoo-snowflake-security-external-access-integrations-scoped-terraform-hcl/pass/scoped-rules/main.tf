resource "snowflake_external_access_integration" "payments_api" {
  name                  = "PAYMENTS_API_ACCESS"
  allowed_network_rules = [snowflake_network_rule.payments_api.fully_qualified_name]
  enabled               = true
  comment               = "Outbound access for the payments UDF"
}
