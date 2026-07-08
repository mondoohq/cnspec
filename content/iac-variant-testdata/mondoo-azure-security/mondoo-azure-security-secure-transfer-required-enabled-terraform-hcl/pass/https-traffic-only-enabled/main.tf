resource "azurerm_storage_account" "pass" {
  name                       = "examplestorage"
  resource_group_name        = "example-rg"
  location                   = "eastus"
  account_tier               = "Standard"
  account_replication_type   = "LRS"
  https_traffic_only_enabled = true
}
