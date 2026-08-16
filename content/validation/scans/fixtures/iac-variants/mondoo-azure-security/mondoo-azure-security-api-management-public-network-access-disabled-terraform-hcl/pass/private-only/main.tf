resource "azurerm_api_management" "internal" {
  name                          = "internal-apim"
  location                      = azurerm_resource_group.prod.location
  resource_group_name           = azurerm_resource_group.prod.name
  publisher_name                = "Example Corp"
  publisher_email               = "platform@example.com"
  sku_name                      = "Developer_1"
  virtual_network_type          = "Internal"
  public_network_access_enabled = false
}
