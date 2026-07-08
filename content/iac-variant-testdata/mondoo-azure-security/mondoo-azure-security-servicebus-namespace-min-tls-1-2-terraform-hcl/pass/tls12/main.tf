resource "azurerm_servicebus_namespace" "pass" {
  name                = "example-namespace"
  location            = "eastus"
  resource_group_name = "example-rg"
  sku                 = "Premium"
  minimum_tls_version = "1.2"
}
