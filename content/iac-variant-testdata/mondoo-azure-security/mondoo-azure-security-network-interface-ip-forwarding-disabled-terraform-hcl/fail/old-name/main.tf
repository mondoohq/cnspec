resource "azurerm_network_interface" "example" {
  name                  = "example-nic"
  location              = "eastus"
  resource_group_name   = "example-rg"
  enable_ip_forwarding  = true

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.example.id
    private_ip_address_allocation = "Dynamic"
  }
}
