resource "azurerm_search_service" "example" {
  name                = "example-search"
  resource_group_name = "rg"
  location            = "eastus"
  sku                 = "standard"
}
