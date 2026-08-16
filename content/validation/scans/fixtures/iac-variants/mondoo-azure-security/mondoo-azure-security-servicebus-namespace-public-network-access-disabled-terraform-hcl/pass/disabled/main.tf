resource "azurerm_servicebus_namespace" "pass" {
  name                          = "example-namespace"
  location                      = "eastus"
  resource_group_name           = "example-rg"
  sku                           = "Premium"
  public_network_access_enabled = false
}
