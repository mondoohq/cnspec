resource "azurerm_mysql_server" "srv" {
  name                = "example-mysql"
  resource_group_name = "example-rg"
  location            = "eastus"
  administrator_login          = "adminuser"
  administrator_login_password = "H@Sh1CoR3!"
  sku_name   = "GP_Gen5_2"
  storage_mb = 5120
  version    = "5.7"
  ssl_minimal_tls_version_enforced = "TLS1_1"
}
