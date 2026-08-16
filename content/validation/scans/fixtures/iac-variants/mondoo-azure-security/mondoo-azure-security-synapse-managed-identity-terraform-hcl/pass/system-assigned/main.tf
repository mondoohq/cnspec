resource "azurerm_synapse_workspace" "analytics" {
  name                                 = "analytics-synapse"
  resource_group_name                  = azurerm_resource_group.prod.name
  location                             = azurerm_resource_group.prod.location
  storage_data_lake_gen2_filesystem_id = azurerm_storage_data_lake_gen2_filesystem.analytics.id
  sql_administrator_login              = "sqladmin"
  sql_administrator_login_password     = var.synapse_password

  identity {
    type = "SystemAssigned"
  }
}
