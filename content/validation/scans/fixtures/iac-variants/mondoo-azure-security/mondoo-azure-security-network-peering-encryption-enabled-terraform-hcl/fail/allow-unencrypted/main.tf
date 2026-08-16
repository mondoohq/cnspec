# AllowUnencrypted means traffic falls back to cleartext whenever a peer does
# not support encryption, which is the case the setting exists to prevent.
resource "azurerm_virtual_network" "prod" {
  name                = "prod-vnet"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  address_space       = ["10.0.0.0/16"]

  encryption {
    enforcement = "AllowUnencrypted"
  }
}
