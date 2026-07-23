resource "azurerm_mysql_flexible_server" "srv" {
  name                   = "example-mysql-flex"
  resource_group_name    = "example-rg"
  location               = "eastus"
  administrator_login    = "adminuser"
  administrator_password = "H@Sh1CoR3!"
  sku_name               = "GP_Standard_D2ds_v4"
  public_network_access_enabled = true
}
