resource "azurerm_postgresql_configuration" "example" {
  name                = "log_checkpoints"
  resource_group_name = "example-rg"
  server_name         = "example-psqlserver"
  value               = "off"
}
