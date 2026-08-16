# Three days of backups is not enough to recover from corruption or ransomware
# that is noticed after a weekend.
resource "azurerm_mysql_server" "app" {
  name                = "app-mysql"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  sku_name            = "GP_Gen5_2"
  version             = "5.7"
  storage_mb          = 51200

  administrator_login          = "mysqladmin"
  administrator_login_password = var.mysql_password

  backup_retention_days   = 3
  ssl_enforcement_enabled = true
}
