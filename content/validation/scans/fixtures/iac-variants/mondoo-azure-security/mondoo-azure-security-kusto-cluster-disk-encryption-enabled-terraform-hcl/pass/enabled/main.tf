resource "azurerm_kusto_cluster" "pass" {
  name                = "examplekusto"
  location            = "eastus"
  resource_group_name = "example-rg"

  sku {
    name     = "Standard_D13_v2"
    capacity = 2
  }

  disk_encryption_enabled = true
}
