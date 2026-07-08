resource "azurerm_synapse_workspace" "example" {
  name                                 = "examplesynapse"
  resource_group_name                  = "example-rg"
  location                             = "eastus"
  storage_data_lake_gen2_filesystem_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Storage/storageAccounts/examplestorageacct/blobServices/default/containers/example"
  sql_administrator_login              = "sqladminuser"
  sql_administrator_login_password     = "H@Sh1CoR3!"

  identity {
    type = "SystemAssigned"
  }

  customer_managed_key {
    key_versionless_id = "https://myvault.vault.azure.net/keys/mykey"
    key_name           = "cmkkey"
  }
}
