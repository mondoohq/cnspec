# Flexible server explicitly disables secure transport.
resource "azurerm_mysql_flexible_server_configuration" "secure_transport" {
  name                = "require_secure_transport"
  resource_group_name = "example-rg"
  server_name         = "example-flexible-mysql"
  value               = "OFF"
}
