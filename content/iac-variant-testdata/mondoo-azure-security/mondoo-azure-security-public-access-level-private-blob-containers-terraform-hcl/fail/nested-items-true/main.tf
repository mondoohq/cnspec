resource "azurerm_storage_account" "example" {
  name                = "examplestoracct"
  resource_group_name = "example-rg"
  location            = "East US"

  account_tier             = "Standard"
  account_replication_type = "GRS"

  allow_nested_items_to_be_public = true
}
