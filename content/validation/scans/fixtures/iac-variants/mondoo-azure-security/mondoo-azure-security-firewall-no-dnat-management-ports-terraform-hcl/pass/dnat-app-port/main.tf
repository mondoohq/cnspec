resource "azurerm_firewall_policy_rule_collection_group" "prod" {
  name               = "prod-rules"
  firewall_policy_id = azurerm_firewall_policy.prod.id
  priority           = 500

  nat_rule_collection {
    name     = "inbound-web"
    priority = 300
    action   = "Dnat"

    rule {
      name                = "web"
      protocols           = ["TCP"]
      source_addresses    = ["*"]
      destination_address = "203.0.113.10"
      destination_ports   = ["443"]
      translated_address  = "10.0.1.10"
      translated_port     = "8443"
    }
  }
}
