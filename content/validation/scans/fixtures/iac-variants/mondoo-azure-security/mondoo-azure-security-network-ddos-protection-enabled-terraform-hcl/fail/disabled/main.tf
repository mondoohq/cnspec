resource "azurerm_network_ddos_protection_plan" "example" {
  name                = "example-ddos"
  location            = "eastus"
  resource_group_name = "example-rg"
}

resource "azurerm_virtual_network" "example" {
  name                = "example-vnet"
  location            = "eastus"
  resource_group_name = "example-rg"
  address_space       = ["10.0.0.0/16"]

  ddos_protection_plan {
    id     = azurerm_network_ddos_protection_plan.example.id
    enable = false
  }
}
