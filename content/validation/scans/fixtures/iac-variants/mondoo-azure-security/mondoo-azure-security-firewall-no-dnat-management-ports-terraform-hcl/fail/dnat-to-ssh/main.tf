# DNAT publishing SSH straight to an internal host, which puts a management
# port on the public internet behind a single NAT rule.
resource "azurerm_firewall_policy_rule_collection_group" "prod" {
  name               = "prod-rules"
  firewall_policy_id = azurerm_firewall_policy.prod.id
  priority           = 500

  nat_rule_collection {
    name     = "inbound-admin"
    priority = 300
    action   = "Dnat"

    rule {
      name                = "ssh-jump"
      protocols           = ["TCP"]
      source_addresses    = ["*"]
      destination_address = "203.0.113.10"
      destination_ports   = ["2222"]
      translated_address  = "10.0.1.5"
      translated_port     = "22"
    }
  }
}
