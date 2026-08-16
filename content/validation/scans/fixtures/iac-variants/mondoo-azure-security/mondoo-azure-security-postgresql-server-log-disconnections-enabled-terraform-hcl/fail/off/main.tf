resource "azurerm_postgresql_configuration" "example" {
  name                = "log_disconnections"
  resource_group_name = "example-rg"
  server_name         = "example-psqlserver"
  value               = "off"
}
