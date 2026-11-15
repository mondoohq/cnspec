resource "azurerm_network_ddos_protection_plan" "example" {
  name                = "example-ddos-plan"
  resource_group_name = "example-rg"
  location            = "East US"
}

resource "azurerm_virtual_network" "example" {
  name                = "example-vnet"
  resource_group_name = "example-rg"
  location            = "East US"
  address_space       = ["10.0.0.0/16"]

  ddos_protection_plan {
    id     = azurerm_network_ddos_protection_plan.example.id
    enable = true
  }
}

resource "azurerm_public_ip" "example" {
  name                = "example-pip"
  resource_group_name = "example-rg"
  location            = "East US"
  allocation_method   = "Static"
  sku                 = "Standard"

  ddos_protection_mode = "VirtualNetworkInherited"
}
