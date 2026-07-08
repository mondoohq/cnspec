resource "azurerm_postgresql_server" "example" {
  name                = "example-psqlserver"
  resource_group_name = "example-rg"
  location            = "eastus"

  administrator_login          = "psqladmin"
  administrator_login_password = "H@Sh1CoR3!"

  sku_name   = "GP_Gen5_4"
  version    = "11"
  storage_mb = 640000

  ssl_enforcement_enabled          = true
  infrastructure_encryption_enabled = true
}
