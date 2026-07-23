resource "azurerm_web_pubsub" "example" {
  name                = "example-webpubsub"
  resource_group_name = "example-rg"
  location            = "eastus"

  sku      = "Standard_S1"
  capacity = 1

  identity {
    type = "SystemAssigned"
  }
}
