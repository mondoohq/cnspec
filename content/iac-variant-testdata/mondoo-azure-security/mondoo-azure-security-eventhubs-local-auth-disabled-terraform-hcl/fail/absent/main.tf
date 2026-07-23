resource "azurerm_eventhub_namespace" "example" {
  name                = "example-ehns"
  location            = "eastus"
  resource_group_name = "example-rg"
  sku                 = "Standard"
}
