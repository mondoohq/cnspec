resource "azurerm_storage_account" "example" {
  name                = "examplestoracct"
  resource_group_name = "example-rg"
  location            = "East US"

  account_tier             = "Standard"
  account_replication_type = "GRS"

  allow_blob_public_access = false
}
