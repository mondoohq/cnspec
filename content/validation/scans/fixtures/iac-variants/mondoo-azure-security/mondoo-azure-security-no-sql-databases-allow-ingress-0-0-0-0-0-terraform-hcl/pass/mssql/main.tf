resource "azurerm_mssql_firewall_rule" "office" {
  name             = "allow-office"
  server_id        = azurerm_mssql_server.example.id
  start_ip_address = "203.0.113.10"
  end_ip_address   = "203.0.113.20"
}
