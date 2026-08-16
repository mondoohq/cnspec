resource "azurerm_mssql_database_extended_auditing_policy" "example" {
  database_id       = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Sql/servers/example-sqlserver/databases/example-db"
  storage_endpoint  = "https://examplestorage.blob.core.windows.net/"
  retention_in_days = 90
  enabled           = true
}
