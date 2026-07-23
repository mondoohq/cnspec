resource "azurerm_iothub" "fail" {
  name                = "example-iothub"
  resource_group_name = "example-rg"
  location            = "eastus"

  sku {
    name     = "S1"
    capacity = 1
  }

  min_tls_version = "1.0"
}
