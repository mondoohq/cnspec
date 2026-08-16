resource "azurerm_recovery_services_vault" "example" {
  name                = "example-recovery-vault"
  location            = "East US"
  resource_group_name = "example-rg"
  sku                 = "Standard"

  soft_delete_enabled = true
}
