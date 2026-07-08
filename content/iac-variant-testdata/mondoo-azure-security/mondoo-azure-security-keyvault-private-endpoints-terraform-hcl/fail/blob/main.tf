resource "azurerm_private_endpoint" "fail" {
  name                = "example-pe"
  location            = "eastus"
  resource_group_name = "example-rg"
  subnet_id           = "/subscriptions/00000000/resourceGroups/example-rg/providers/Microsoft.Network/virtualNetworks/example-vnet/subnets/example-subnet"

  private_service_connection {
    name                           = "example-psc"
    private_connection_resource_id = "/subscriptions/00000000/resourceGroups/example-rg/providers/Microsoft.Storage/storageAccounts/examplesa"
    subresource_names              = ["blob"]
    is_manual_connection           = false
  }
}
