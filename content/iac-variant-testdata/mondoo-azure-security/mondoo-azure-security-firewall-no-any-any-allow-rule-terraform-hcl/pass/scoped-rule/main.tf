resource "azurerm_firewall_policy_rule_collection_group" "prod" {
  name               = "prod-rules"
  firewall_policy_id = azurerm_firewall_policy.prod.id
  priority           = 500

  network_rule_collection {
    name     = "allow-app-to-db"
    priority = 400
    action   = "Allow"

    rule {
      name                  = "app-to-sql"
      protocols             = ["TCP"]
      source_addresses      = ["10.0.1.0/24"]
      destination_addresses = ["10.0.2.10"]
      destination_ports     = ["1433"]
    }
  }
}
