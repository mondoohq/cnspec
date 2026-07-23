resource "azurerm_postgresql_firewall_rule" "example" {
  name                = "office"
  resource_group_name = "example-rg"
  server_name         = "example-psqlserver"
  start_ip_address    = "40.112.8.12"
  end_ip_address      = "40.112.8.20"
}
