resource "azurerm_mssql_firewall_rule" "allow_all" {
  name             = "allow-all-azure"
  server_id        = azurerm_mssql_server.example.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0"
}
