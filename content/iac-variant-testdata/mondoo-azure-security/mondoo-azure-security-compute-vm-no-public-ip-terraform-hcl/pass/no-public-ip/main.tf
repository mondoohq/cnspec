resource "azurerm_network_interface" "example" {
  name                = "nic-example"
  location            = "eastus"
  resource_group_name = "example-rg"

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.example.id
    private_ip_address_allocation = "Dynamic"
  }
}
