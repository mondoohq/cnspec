resource "azurerm_synapse_workspace" "example" {
  name                                 = "examplesynapse"
  resource_group_name                  = "example-rg"
  location                             = "eastus"
  storage_data_lake_gen2_filesystem_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Storage/storageAccounts/examplestorageacct/blobServices/default/containers/example"
  sql_administrator_login              = "sqladminuser"
  sql_administrator_login_password     = "H@Sh1CoR3!"
  azuread_authentication_only          = false

  identity {
    type = "SystemAssigned"
  }
}
