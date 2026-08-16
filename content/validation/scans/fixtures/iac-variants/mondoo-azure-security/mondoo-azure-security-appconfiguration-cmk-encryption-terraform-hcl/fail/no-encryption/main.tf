resource "azurerm_app_configuration" "fail" {
  name                = "example-appconf"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "standard"
}
