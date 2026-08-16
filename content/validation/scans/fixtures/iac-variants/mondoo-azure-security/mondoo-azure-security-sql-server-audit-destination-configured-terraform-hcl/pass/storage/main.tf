resource "azurerm_mssql_server_extended_auditing_policy" "example" {
  server_id                  = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Sql/servers/example-sqlserver"
  storage_endpoint           = "https://examplestorage.blob.core.windows.net/"
  storage_account_access_key = "examplekey=="
  retention_in_days          = 90
  log_monitoring_enabled     = false
}
