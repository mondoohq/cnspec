resource "azurerm_firewall_policy" "prod" {
  name                = "prod-policy"
  resource_group_name = azurerm_resource_group.prod.name
  location            = azurerm_resource_group.prod.location
  sku                 = "Premium"

  intrusion_detection {
    mode = "Deny"
  }
}
