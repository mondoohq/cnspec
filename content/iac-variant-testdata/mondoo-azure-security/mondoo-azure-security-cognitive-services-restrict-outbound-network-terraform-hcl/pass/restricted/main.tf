resource "azurerm_cognitive_account" "vision" {
  name                               = "vision"
  location                           = azurerm_resource_group.prod.location
  resource_group_name                = azurerm_resource_group.prod.name
  kind                               = "ComputerVision"
  sku_name                           = "S1"
  outbound_network_access_restricted = true
  fqdns                              = ["storage.example.com"]
}
