resource "azurerm_subnet" "example" {
  name                            = "internal"
  resource_group_name             = "example-rg"
  virtual_network_name            = "example-vnet"
  address_prefixes                = ["10.0.1.0/24"]
  default_outbound_access_enabled = true
}
