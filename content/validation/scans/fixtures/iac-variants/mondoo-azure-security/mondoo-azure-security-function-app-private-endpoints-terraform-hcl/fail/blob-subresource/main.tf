resource "azurerm_private_endpoint" "example" {
  name                = "example-storage-pe"
  location            = "eastus"
  resource_group_name = "example-rg"
  subnet_id           = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/virtualNetworks/example-vnet/subnets/pe-subnet"

  private_service_connection {
    name                           = "example-storage-psc"
    private_connection_resource_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Storage/storageAccounts/examplestorage"
    subresource_names              = ["blob"]
    is_manual_connection           = false
  }
}
