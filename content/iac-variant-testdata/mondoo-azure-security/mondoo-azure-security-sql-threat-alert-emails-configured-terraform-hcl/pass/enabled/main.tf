resource "azurerm_mssql_server_security_alert_policy" "example" {
  resource_group_name = "example-rg"
  server_name         = azurerm_mssql_server.example.name
  state               = "Enabled"
  email_addresses     = ["secops@example.com", "dba@example.com"]
  retention_days      = 30
}
