resource "azurerm_postgresql_configuration" "example" {
  name                = "log_connections"
  resource_group_name = "example-rg"
  server_name         = "example-psqlserver"
  value               = "off"
}
