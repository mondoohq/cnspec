resource "azurerm_mysql_server" "app" {
  name                = "app-mysql"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  sku_name            = "GP_Gen5_2"
  version             = "5.7"
  storage_mb          = 51200

  administrator_login          = "mysqladmin"
  administrator_login_password = var.mysql_password

  backup_retention_days        = 30
  geo_redundant_backup_enabled = true
  ssl_enforcement_enabled      = true
}
