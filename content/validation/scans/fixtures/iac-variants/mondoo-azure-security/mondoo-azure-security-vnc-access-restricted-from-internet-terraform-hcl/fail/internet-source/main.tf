resource "azurerm_network_security_group" "example" {
  name                = "example-nsg"
  resource_group_name = "example-rg"
  location            = "eastus"

  security_rule {
    name                       = "allow-vnc-internet"
    priority                   = 130
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "5901"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }
}
