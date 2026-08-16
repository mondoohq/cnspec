resource "azurerm_private_endpoint" "pass" {
  name                = "example-pe"
  location            = "eastus"
  resource_group_name = "example-rg"
  subnet_id           = "/subscriptions/00000000/resourceGroups/example-rg/providers/Microsoft.Network/virtualNetworks/example-vnet/subnets/example-subnet"

  private_service_connection {
    name                           = "example-psc"
    private_connection_resource_id = "/subscriptions/00000000/resourceGroups/example-rg/providers/Microsoft.KeyVault/vaults/example-kv"
    subresource_names              = ["vault"]
    is_manual_connection           = false
  }
}
