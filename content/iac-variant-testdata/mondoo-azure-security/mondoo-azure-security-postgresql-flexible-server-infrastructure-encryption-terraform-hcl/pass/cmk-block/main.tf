resource "azurerm_postgresql_flexible_server" "example" {
  name                = "example-psqlflexibleserver"
  resource_group_name = "example-rg"
  location            = "eastus"
  version             = "13"
  storage_mb          = 32768
  sku_name            = "GP_Standard_D4s_v3"

  identity {
    type         = "UserAssigned"
    identity_ids = ["/subscriptions/000/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/uai"]
  }

  customer_managed_key {
    key_vault_key_id                  = "https://myvault.vault.azure.net/keys/mykey/version"
    primary_user_assigned_identity_id = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/uai"
  }
}
