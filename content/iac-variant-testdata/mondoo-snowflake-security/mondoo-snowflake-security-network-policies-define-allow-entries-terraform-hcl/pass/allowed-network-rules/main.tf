# Network rules are the newer way to express the same allow list.
resource "snowflake_network_policy" "corporate" {
  name                     = "CORPORATE_ACCESS"
  allowed_network_rule_list = [snowflake_network_rule.corp_egress.qualified_name]
  comment                  = "Office egress ranges, expressed as a network rule"
}
