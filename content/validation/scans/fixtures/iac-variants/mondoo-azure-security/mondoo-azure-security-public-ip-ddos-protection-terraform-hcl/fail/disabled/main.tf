resource "azurerm_public_ip" "example" {
  name                = "example-pip"
  resource_group_name = "example-rg"
  location            = "East US"
  allocation_method   = "Static"
  sku                 = "Standard"

  ddos_protection_mode = "Disabled"
}
