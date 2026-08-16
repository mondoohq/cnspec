resource "azurerm_recovery_services_vault" "example" {
  name                = "example-recovery-vault"
  location            = "East US"
  resource_group_name = "example-rg"
  sku                 = "Standard"

  soft_delete_enabled = true
}

resource "azurerm_backup_policy_vm" "example" {
  name                = "example-vm-backup-policy"
  resource_group_name = "example-rg"
  recovery_vault_name = azurerm_recovery_services_vault.example.name

  backup {
    frequency = "Daily"
    time      = "23:00"
  }

  retention_daily {
    count = 10
  }
}
