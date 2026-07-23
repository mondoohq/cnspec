resource "azurerm_iothub" "pass" {
  name                = "example-iothub"
  resource_group_name = "example-rg"
  location            = "eastus"

  sku {
    name     = "S1"
    capacity = 1
  }

  public_network_access_enabled = false
}
