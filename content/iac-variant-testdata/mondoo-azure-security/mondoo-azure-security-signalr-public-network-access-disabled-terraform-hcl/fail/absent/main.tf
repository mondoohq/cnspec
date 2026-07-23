resource "azurerm_signalr_service" "example" {
  name                = "example-signalr"
  location            = "eastus"
  resource_group_name = "example-rg"

  sku {
    name     = "Standard_S1"
    capacity = 1
  }
}
