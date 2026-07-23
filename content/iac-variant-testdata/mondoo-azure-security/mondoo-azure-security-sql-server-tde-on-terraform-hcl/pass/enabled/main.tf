resource "azurerm_mssql_database" "example" {
  name                                = "example-db"
  server_id                           = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Sql/servers/example-sqlserver"
  transparent_data_encryption_enabled = true
}
