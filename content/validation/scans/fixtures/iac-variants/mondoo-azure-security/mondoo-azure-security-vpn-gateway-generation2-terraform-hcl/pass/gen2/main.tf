resource "azurerm_virtual_network_gateway" "example" {
  name                = "example-vpn-gw"
  resource_group_name = "example-rg"
  location            = "eastus"

  type     = "Vpn"
  vpn_type = "RouteBased"
  sku      = "VpnGw2"
  generation = "Generation2"

  ip_configuration {
    name                          = "vnetGatewayConfig"
    public_ip_address_id          = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/publicIPAddresses/example-pip"
    private_ip_address_allocation = "Dynamic"
    subnet_id                     = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/virtualNetworks/example-vnet/subnets/GatewaySubnet"
  }
}
