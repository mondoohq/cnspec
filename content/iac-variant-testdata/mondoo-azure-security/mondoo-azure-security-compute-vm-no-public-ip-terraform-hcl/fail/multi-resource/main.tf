resource "azurerm_network_interface" "compliant" {
  name                = "nic-compliant"
  location            = "eastus"
  resource_group_name = "example-rg"

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.example.id
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_network_interface" "violating" {
  name                = "nic-violating"
  location            = "eastus"
  resource_group_name = "example-rg"

  ip_configuration {
    name                          = "external"
    subnet_id                     = azurerm_subnet.example.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.example.id
  }
}
