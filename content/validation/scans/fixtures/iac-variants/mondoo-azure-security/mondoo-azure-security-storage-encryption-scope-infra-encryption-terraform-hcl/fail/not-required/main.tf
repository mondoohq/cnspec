resource "azurerm_storage_encryption_scope" "example" {
  name                               = "cmkscope"
  storage_account_id                 = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Storage/storageAccounts/examplestorageacct"
  source                             = "Microsoft.Storage"
  infrastructure_encryption_required = false
}
