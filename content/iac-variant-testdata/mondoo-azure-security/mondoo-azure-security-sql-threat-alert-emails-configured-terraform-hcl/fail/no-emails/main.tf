resource "azurerm_mssql_server_security_alert_policy" "example" {
  resource_group_name = "example-rg"
  server_name         = azurerm_mssql_server.example.name
  state               = "Enabled"
}
