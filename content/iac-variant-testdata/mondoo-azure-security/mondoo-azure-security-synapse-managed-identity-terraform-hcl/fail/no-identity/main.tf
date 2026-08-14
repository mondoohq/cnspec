# Without a managed identity the workspace authenticates to linked data stores
# with keys or SAS tokens held in its configuration.
resource "azurerm_synapse_workspace" "analytics" {
  name                                 = "analytics-synapse"
  resource_group_name                  = azurerm_resource_group.prod.name
  location                             = azurerm_resource_group.prod.location
  storage_data_lake_gen2_filesystem_id = azurerm_storage_data_lake_gen2_filesystem.analytics.id
  sql_administrator_login              = "sqladmin"
  sql_administrator_login_password     = var.synapse_password
}
