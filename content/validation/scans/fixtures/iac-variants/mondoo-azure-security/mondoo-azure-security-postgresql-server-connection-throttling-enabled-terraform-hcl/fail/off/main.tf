resource "azurerm_postgresql_configuration" "example" {
  name                = "connection_throttling"
  resource_group_name = "example-rg"
  server_name         = "example-psqlserver"
  value               = "off"
}
