resource "azurerm_mysql_server" "example" {
  name                = "example-mysql"
  resource_group_name = "example-rg"
  location            = "eastus"

  administrator_login          = "mysqladmin"
  administrator_login_password = "P@ssw0rd1234!"

  sku_name   = "GP_Gen5_2"
  storage_mb = 5120
  version    = "5.7"

  ssl_enforcement_enabled = false
}
