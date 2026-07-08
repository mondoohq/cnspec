resource "azurerm_mysql_firewall_rule" "fail" {
  name                = "AllowAll"
  resource_group_name = "example-rg"
  server_name         = "example-mysql"
  start_ip_address    = "0.0.0.0"
  end_ip_address      = "255.255.255.255"
}
