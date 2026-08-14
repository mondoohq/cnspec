# With no network_rule_set the hub accepts device and service connections from
# any network.
resource "azurerm_iothub" "fleet" {
  name                = "fleet-hub"
  resource_group_name = azurerm_resource_group.prod.name
  location            = azurerm_resource_group.prod.location

  sku {
    name     = "S1"
    capacity = 1
  }
}
