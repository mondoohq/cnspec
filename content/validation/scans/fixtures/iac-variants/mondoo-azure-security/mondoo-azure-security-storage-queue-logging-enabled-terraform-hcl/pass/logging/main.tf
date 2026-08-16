resource "azurerm_storage_account" "example" {
  name                     = "examplestorageacct"
  resource_group_name      = "example-rg"
  location                 = "eastus"
  account_tier             = "Standard"
  account_replication_type = "LRS"

  queue_properties {
    logging {
      read                  = true
      write                 = true
      delete                = true
      version               = "1.0"
      retention_policy_days = 10
    }
  }
}
