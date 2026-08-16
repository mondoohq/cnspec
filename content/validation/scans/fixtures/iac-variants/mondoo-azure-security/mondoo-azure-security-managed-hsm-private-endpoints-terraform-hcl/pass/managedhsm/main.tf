resource "azurerm_private_endpoint" "example" {
  name                = "example-hsm-pe"
  location            = "eastus"
  resource_group_name = "example-rg"
  subnet_id           = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/virtualNetworks/example-vnet/subnets/example-subnet"

  private_service_connection {
    name                           = "example-hsm-connection"
    private_connection_resource_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.KeyVault/managedHSMs/example-hsm"
    is_manual_connection           = false
    subresource_names              = ["managedhsm"]
  }
}
