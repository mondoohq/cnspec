resource "azurerm_postgresql_flexible_server_configuration" "throttle" {
  name      = "connection_throttle.enable"
  server_id = azurerm_postgresql_flexible_server.example.id
  value     = "off"
}
