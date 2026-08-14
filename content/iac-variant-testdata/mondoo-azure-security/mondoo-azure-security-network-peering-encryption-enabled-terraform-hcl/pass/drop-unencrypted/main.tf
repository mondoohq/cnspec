resource "azurerm_virtual_network" "prod" {
  name                = "prod-vnet"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  address_space       = ["10.0.0.0/16"]

  encryption {
    enforcement = "DropUnencrypted"
  }

  subnet {
    name             = "app"
    address_prefixes = ["10.0.1.0/24"]
  }
}
