resource "azurerm_network_ddos_protection_plan" "example" {
  name                = "example-ddos-plan"
  resource_group_name = "example-rg"
  location            = "East US"
}

resource "azurerm_virtual_network" "protected" {
  name                = "protected-vnet"
  resource_group_name = "example-rg"
  location            = "East US"
  address_space       = ["10.0.0.0/16"]

  ddos_protection_plan {
    id     = azurerm_network_ddos_protection_plan.example.id
    enable = true
  }
}

resource "azurerm_virtual_network" "unprotected" {
  name                = "unprotected-vnet"
  resource_group_name = "example-rg"
  location            = "East US"
  address_space       = ["10.1.0.0/16"]
}

resource "azurerm_subnet" "unprotected" {
  name                 = "workload"
  resource_group_name  = "example-rg"
  virtual_network_name = azurerm_virtual_network.unprotected.name
  address_prefixes     = ["10.1.1.0/24"]
}

resource "azurerm_public_ip" "example" {
  name                = "example-pip"
  resource_group_name = "example-rg"
  location            = "East US"
  allocation_method   = "Static"
  sku                 = "Standard"

  ddos_protection_mode = "VirtualNetworkInherited"
}

resource "azurerm_network_interface" "example" {
  name                = "example-nic"
  resource_group_name = "example-rg"
  location            = "East US"

  ip_configuration {
    name                          = "primary"
    subnet_id                     = azurerm_subnet.unprotected.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.example.id
  }
}
