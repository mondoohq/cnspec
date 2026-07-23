resource "azurerm_mssql_server_microsoft_support_auditing_policy" "example" {
  server_id                  = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Sql/servers/example-sqlserver"
  blob_storage_endpoint      = "https://examplestorage.blob.core.windows.net/"
  storage_account_access_key = "examplekey=="
  enabled                    = true
}
