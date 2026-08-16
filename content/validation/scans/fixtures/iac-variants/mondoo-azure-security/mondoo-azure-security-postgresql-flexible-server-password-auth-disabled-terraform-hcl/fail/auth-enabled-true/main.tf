resource "azurerm_postgresql_flexible_server" "example" {
  name                = "example-psqlflexibleserver"
  resource_group_name = "example-rg"
  location            = "eastus"
  version             = "13"
  storage_mb          = 32768
  sku_name            = "GP_Standard_D4s_v3"

  administrator_login    = "psqladmin"
  administrator_password = "H@Sh1CoR3!"

  authentication {
    password_auth_enabled         = true
    active_directory_auth_enabled = true
    tenant_id                     = "00000000-0000-0000-0000-000000000000"
  }
}
