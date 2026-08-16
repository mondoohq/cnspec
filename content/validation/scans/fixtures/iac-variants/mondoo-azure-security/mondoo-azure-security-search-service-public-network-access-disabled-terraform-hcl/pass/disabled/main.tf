resource "azurerm_search_service" "pass" {
  name                          = "example-search"
  resource_group_name           = "example-rg"
  location                      = "eastus"
  sku                           = "standard"
  public_network_access_enabled = false
}
