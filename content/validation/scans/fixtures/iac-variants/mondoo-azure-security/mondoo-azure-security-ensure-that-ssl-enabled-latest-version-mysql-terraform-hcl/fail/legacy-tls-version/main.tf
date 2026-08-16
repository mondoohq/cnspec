# Secure transport is on, but the server still accepts TLS 1.1.
resource "azurerm_mysql_flexible_server_configuration" "secure_transport" {
  name                = "require_secure_transport"
  resource_group_name = "example-rg"
  server_name         = "example-flexible-mysql"
  value               = "ON"
}

resource "azurerm_mysql_flexible_server_configuration" "tls_version" {
  name                = "tls_version"
  resource_group_name = "example-rg"
  server_name         = "example-flexible-mysql"
  value               = "TLSv1.1,TLSv1.2"
}
