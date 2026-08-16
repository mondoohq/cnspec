resource "azurerm_mssql_server_extended_auditing_policy" "example" {
  server_id              = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Sql/servers/example-sqlserver"
  log_monitoring_enabled = false
}
