resource "azurerm_synapse_workspace" "example" {
  name                                 = "examplesynapse"
  resource_group_name                  = "example-rg"
  location                             = "eastus"
  storage_data_lake_gen2_filesystem_id = "https://examplestorage.dfs.core.windows.net/example"
  sql_administrator_login              = "sqladminuser"
  sql_administrator_login_password     = "H@Sh1CoR3!"

  public_network_access_enabled = true

  identity {
    type = "SystemAssigned"
  }
}
