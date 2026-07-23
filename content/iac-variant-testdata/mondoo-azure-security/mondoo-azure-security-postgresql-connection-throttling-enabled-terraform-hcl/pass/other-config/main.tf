resource "azurerm_postgresql_flexible_server_configuration" "logging" {
  name      = "log_connections"
  server_id = azurerm_postgresql_flexible_server.example.id
  value     = "on"
}
