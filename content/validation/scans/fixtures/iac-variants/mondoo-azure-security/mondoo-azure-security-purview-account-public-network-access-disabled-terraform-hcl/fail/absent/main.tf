resource "azurerm_purview_account" "example" {
  name                = "example-purview"
  resource_group_name = "example-rg"
  location            = "East US"

  identity {
    type = "SystemAssigned"
  }
}
