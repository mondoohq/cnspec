resource "azurerm_storage_account" "example" {
  name                            = "examplestorageacct"
  resource_group_name             = "example-rg"
  location                        = "eastus"
  account_tier                    = "Standard"
  account_replication_type        = "GRS"
  cross_tenant_replication_enabled = true
}
