resource "azurerm_search_service" "pass" {
  name                                     = "example-search"
  resource_group_name                      = "example-rg"
  location                                 = "eastus"
  sku                                      = "standard"
  customer_managed_key_enforcement_enabled = true
}
