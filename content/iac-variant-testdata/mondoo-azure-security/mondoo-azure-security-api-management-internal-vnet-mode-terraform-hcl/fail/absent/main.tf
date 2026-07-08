resource "azurerm_api_management" "example" {
  name                = "example-apim"
  location            = "eastus"
  resource_group_name = "example-rg"
  publisher_name      = "Example"
  publisher_email     = "admin@example.com"
  sku_name            = "Developer_1"
}
