# UDP/53 (DNS) exposed to the internet is one of the explicitly flagged ports.
resource "azurerm_network_security_group" "example" {
  name                = "example-nsg"
  location            = "eastus"
  resource_group_name = "example-rg"

  security_rule {
    name                       = "allow-dns-udp"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Udp"
    source_port_range          = "*"
    destination_port_range     = "53"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }
}
