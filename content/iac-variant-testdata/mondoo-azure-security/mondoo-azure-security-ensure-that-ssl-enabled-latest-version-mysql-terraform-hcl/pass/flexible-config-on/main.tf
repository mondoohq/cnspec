# Flexible server enforces secure transport via server parameter set to ON.
resource "azurerm_mysql_flexible_server_configuration" "secure_transport" {
  name                = "require_secure_transport"
  resource_group_name = "example-rg"
  server_name         = "example-flexible-mysql"
  value               = "ON"
}
