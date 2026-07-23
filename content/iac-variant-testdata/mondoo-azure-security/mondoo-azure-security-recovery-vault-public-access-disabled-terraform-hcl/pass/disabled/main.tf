resource "azurerm_recovery_services_vault" "pass" {
  name                          = "example-vault"
  location                      = "eastus"
  resource_group_name           = "example-rg"
  sku                           = "Standard"
  public_network_access_enabled = false
}
