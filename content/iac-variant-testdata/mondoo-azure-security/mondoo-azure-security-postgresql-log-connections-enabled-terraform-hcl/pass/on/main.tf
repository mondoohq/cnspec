resource "azurerm_postgresql_flexible_server_configuration" "example" {
  name      = "log_connections"
  server_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/example-pg"
  value     = "on"
}
