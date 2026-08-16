resource "azurerm_container_registry" "pass" {
  name                          = "examplereg"
  resource_group_name           = "example-rg"
  location                      = "eastus"
  sku                           = "Premium"
  public_network_access_enabled = false
}
