resource "azurerm_mysql_firewall_rule" "corporate" {
  name                = "corporate-egress"
  resource_group_name = azurerm_resource_group.prod.name
  server_name         = azurerm_mysql_server.app.name
  start_ip_address    = "203.0.113.0"
  end_ip_address      = "203.0.113.255"
}
