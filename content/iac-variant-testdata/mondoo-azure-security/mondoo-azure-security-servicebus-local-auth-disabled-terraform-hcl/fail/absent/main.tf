resource "azurerm_servicebus_namespace" "fail" {
  name                = "example-namespace"
  location            = "eastus"
  resource_group_name = "example-rg"
  sku                 = "Premium"
}
