resource "azurerm_data_factory" "example" {
  name                = "example-adf"
  location            = "eastus"
  resource_group_name = "example-rg"

  identity {
    type = "SystemAssigned"
  }
}
