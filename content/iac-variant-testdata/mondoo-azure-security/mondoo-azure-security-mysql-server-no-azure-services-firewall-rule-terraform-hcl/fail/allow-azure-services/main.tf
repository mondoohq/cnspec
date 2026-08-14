# 0.0.0.0 to 0.0.0.0 is the "Allow access to Azure services" rule. It admits
# every Azure tenant, not just this subscription.
resource "azurerm_mysql_firewall_rule" "azure_services" {
  name                = "AllowAllWindowsAzureIps"
  resource_group_name = azurerm_resource_group.prod.name
  server_name         = azurerm_mysql_server.app.name
  start_ip_address    = "0.0.0.0"
  end_ip_address      = "0.0.0.0"
}
