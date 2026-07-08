resource "azurerm_postgresql_firewall_rule" "example" {
  name                = "allow-all"
  resource_group_name = "example-rg"
  server_name         = "example-psqlserver"
  start_ip_address    = "0.0.0.0"
  end_ip_address      = "255.255.255.255"
}
