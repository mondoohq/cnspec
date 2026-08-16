resource "azurerm_mysql_firewall_rule" "pass" {
  name                = "office"
  resource_group_name = "example-rg"
  server_name         = "example-mysql"
  start_ip_address    = "203.0.113.0"
  end_ip_address      = "203.0.113.255"
}
