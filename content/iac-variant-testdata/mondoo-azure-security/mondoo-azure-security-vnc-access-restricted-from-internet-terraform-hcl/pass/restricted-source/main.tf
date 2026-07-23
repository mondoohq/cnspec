resource "azurerm_network_security_group" "example" {
  name                = "example-nsg"
  resource_group_name = "example-rg"
  location            = "eastus"

  security_rule {
    name                       = "allow-vnc-corp"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "5900"
    source_address_prefix      = "203.0.113.5"
    destination_address_prefix = "*"
  }
}
