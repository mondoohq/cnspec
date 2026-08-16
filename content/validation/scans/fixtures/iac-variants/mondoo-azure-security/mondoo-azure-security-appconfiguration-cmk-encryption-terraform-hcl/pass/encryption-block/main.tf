resource "azurerm_app_configuration" "pass" {
  name                = "example-appconf"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "standard"

  identity {
    type         = "UserAssigned"
    identity_ids = ["/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/example-id"]
  }

  encryption {
    key_vault_key_identifier = "https://example-kv.vault.azure.net/keys/example-key/abc123"
    identity_client_id       = "11111111-1111-1111-1111-111111111111"
  }
}
