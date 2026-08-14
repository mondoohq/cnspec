# Any source to any destination, allowed. A rule like this makes every other
# rule in the policy irrelevant.
resource "azurerm_firewall_policy_rule_collection_group" "prod" {
  name               = "prod-rules"
  firewall_policy_id = azurerm_firewall_policy.prod.id
  priority           = 500

  network_rule_collection {
    name     = "allow-all"
    priority = 400
    action   = "Allow"

    rule {
      name                  = "any-any"
      protocols             = ["Any"]
      source_addresses      = ["*"]
      destination_addresses = ["*"]
      destination_ports     = ["*"]
    }
  }
}
