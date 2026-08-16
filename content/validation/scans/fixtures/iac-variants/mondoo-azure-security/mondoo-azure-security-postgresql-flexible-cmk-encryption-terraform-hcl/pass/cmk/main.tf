resource "azurerm_postgresql_flexible_server" "example" {
  name                = "example-psqlflexibleserver"
  resource_group_name = "example-rg"
  location            = "eastus"
  version             = "16"
  storage_mb          = 32768
  sku_name            = "GP_Standard_D2s_v3"

  administrator_login    = "psqladmin"
  administrator_password = "H@Sh1CoR3!"

  customer_managed_key {
    key_vault_key_id                  = azurerm_key_vault_key.example.id
    primary_user_assigned_identity_id = azurerm_user_assigned_identity.example.id
  }

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.example.id]
  }
}
