resource "azurerm_signalr_service" "fail" {
  name                = "example-signalr"
  location            = "eastus"
  resource_group_name = "example-rg"
  local_auth_enabled  = true

  sku {
    name     = "Standard_S1"
    capacity = 1
  }
}
