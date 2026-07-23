resource "azurerm_network_security_group" "example" {
  name                = "example-nsg"
  resource_group_name = "example-rg"
  location            = "eastus"

  security_rule {
    name                       = "allow-https"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }
}
