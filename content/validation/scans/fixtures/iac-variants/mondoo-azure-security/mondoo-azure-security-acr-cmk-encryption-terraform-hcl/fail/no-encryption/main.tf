resource "azurerm_container_registry" "fail" {
  name                = "examplereg"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "Premium"
}
