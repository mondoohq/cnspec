resource "azurerm_network_security_group" "db" {
  name                = "db-nsg"
  location            = "eastus"
  resource_group_name = "example-rg"

  security_rule {
    name                       = "allow-mssql"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "1433"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}
