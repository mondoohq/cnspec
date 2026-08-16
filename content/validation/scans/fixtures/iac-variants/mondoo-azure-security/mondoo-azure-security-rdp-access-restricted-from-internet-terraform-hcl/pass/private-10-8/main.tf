resource "azurerm_network_security_group" "example" {
  name                = "example-nsg"
  location            = "East US"
  resource_group_name = "example-rg"

  security_rule {
    name                       = "allow-rdp-corp"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "3389"
    source_address_prefix      = "10.0.0.0/8"
    destination_address_prefix = "*"
  }
}
