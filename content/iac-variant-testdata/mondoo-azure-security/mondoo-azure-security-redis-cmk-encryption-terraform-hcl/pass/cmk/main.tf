resource "azurerm_managed_redis" "example" {
  name                = "example-managed-redis"
  location            = "eastus"
  resource_group_name = "example-resources"
  sku_name            = "Balanced_B0"

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.example.id]
  }

  customer_managed_key {
    key_vault_key_id          = azurerm_key_vault_key.example.id
    user_assigned_identity_id = azurerm_user_assigned_identity.example.id
  }
}
