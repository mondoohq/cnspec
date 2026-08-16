resource "azurerm_data_factory" "example" {
  name                   = "example-adf"
  location               = "eastus"
  resource_group_name    = "example-rg"
  public_network_enabled = true

  identity {
    type = "SystemAssigned"
  }
}
