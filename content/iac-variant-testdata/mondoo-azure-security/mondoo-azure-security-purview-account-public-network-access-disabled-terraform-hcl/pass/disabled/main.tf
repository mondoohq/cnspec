resource "azurerm_purview_account" "example" {
  name                = "example-purview"
  resource_group_name = "example-rg"
  location            = "East US"

  public_network_enabled = false

  identity {
    type = "SystemAssigned"
  }
}
