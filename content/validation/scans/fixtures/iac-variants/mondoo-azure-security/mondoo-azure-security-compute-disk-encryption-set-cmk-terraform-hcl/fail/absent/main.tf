resource "azurerm_disk_encryption_set" "example" {
  name                = "des-example"
  resource_group_name = "example-rg"
  location            = "eastus"

  identity {
    type = "SystemAssigned"
  }
}
