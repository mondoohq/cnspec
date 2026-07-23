resource "azurerm_postgresql_firewall_rule" "example" {
  name                = "single-host"
  resource_group_name = "example-rg"
  server_name         = "example-psqlserver"
  start_ip_address    = "52.168.1.1"
  end_ip_address      = "52.168.1.1"
}
