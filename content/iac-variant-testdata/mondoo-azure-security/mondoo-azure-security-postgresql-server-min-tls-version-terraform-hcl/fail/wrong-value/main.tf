resource "azurerm_postgresql_server" "example" {
  name                = "example-psqlserver"
  location            = "East US"
  resource_group_name = "example-rg"

  sku_name   = "GP_Gen5_4"
  version    = "11"
  storage_mb = 640000

  administrator_login          = "psqladmin"
  administrator_login_password = "H@Sh1CoR3!"

  ssl_enforcement_enabled          = true
  ssl_minimal_tls_version_enforced = "TLS1_1"
  public_network_access_enabled    = false
}
