# Any source may reach any FQDN over HTTPS.
resource "azurerm_firewall_policy_rule_collection_group" "prod" {
  name               = "prod-rules"
  firewall_policy_id = azurerm_firewall_policy.prod.id
  priority           = 500

  application_rule_collection {
    name     = "allow-web"
    priority = 400
    action   = "Allow"

    rule {
      name = "any-web"
      protocols {
        type = "Https"
        port = 443
      }
      source_addresses  = ["*"]
      destination_fqdns = ["*"]
    }
  }
}
