resource "azurerm_mssql_server_extended_auditing_policy" "example" {
  server_id                  = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Sql/servers/example"
  storage_endpoint           = "https://examplesa.blob.core.windows.net/"
  storage_account_access_key = "examplekey"
  retention_in_days          = 15
}
