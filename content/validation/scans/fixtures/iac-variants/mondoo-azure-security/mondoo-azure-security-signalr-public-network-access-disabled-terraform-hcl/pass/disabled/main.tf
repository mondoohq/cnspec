resource "azurerm_signalr_service" "example" {
  name                          = "example-signalr"
  location                      = "eastus"
  resource_group_name           = "example-rg"
  public_network_access_enabled = false

  sku {
    name     = "Standard_S1"
    capacity = 1
  }
}
