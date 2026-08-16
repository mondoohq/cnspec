resource "azurerm_network_security_rule" "mysql_from_subnet" {
  name                        = "allow-mysql-internal"
  priority                    = 110
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "3306"
  source_address_prefix       = "10.0.1.0/24"
  destination_address_prefix  = "*"
  resource_group_name         = "example-rg"
  network_security_group_name = "example-nsg"
}
