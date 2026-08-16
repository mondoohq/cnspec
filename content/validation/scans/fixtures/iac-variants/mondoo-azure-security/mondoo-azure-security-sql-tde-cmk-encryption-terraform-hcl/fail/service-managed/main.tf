# Service-managed key: no customer key vault key -> not CMK encrypted
resource "azurerm_mssql_server_transparent_data_encryption" "example" {
  server_id = azurerm_mssql_server.example.id
}
