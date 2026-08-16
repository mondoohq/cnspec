resource "azurerm_storage_account" "example" {
  name                     = "examplestorageacct"
  resource_group_name      = "example-rg"
  location                 = "eastus"
  account_tier             = "Standard"
  account_replication_type = "LRS"

  queue_properties {
    hour_metrics {
      enabled               = true
      version               = "1.0"
      include_apis          = true
      retention_policy_days = 10
    }
  }
}
