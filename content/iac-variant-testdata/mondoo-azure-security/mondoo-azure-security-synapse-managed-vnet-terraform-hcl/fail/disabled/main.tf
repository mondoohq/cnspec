resource "azurerm_synapse_workspace" "example" {
  name                                 = "examplesynapse"
  resource_group_name                  = "example-rg"
  location                             = "eastus"
  storage_data_lake_gen2_filesystem_id = "https://examplestorage.dfs.core.windows.net/example"
  sql_administrator_login              = "sqladminuser"
  sql_administrator_login_password     = "H@Sh1CoR3!"

  managed_virtual_network_enabled = false

  identity {
    type = "SystemAssigned"
  }
}
