resource "azurerm_network_security_group" "example" {
  name                = "example-nsg"
  location            = "eastus"
  resource_group_name = "example-rg"

  security_rule {
    name                       = "allow-ssh-internet"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }
}
