resource "azurerm_recovery_services_vault" "fail" {
  name                = "example-vault"
  location            = "eastus"
  resource_group_name = "example-rg"
  sku                 = "Standard"

  monitoring {
    alerts_for_all_job_failures_enabled = false
  }
}
