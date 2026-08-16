resource "azurerm_kusto_cluster" "fail" {
  name                = "examplekusto"
  location            = "eastus"
  resource_group_name = "example-rg"

  sku {
    name     = "Standard_D13_v2"
    capacity = 2
  }
}
