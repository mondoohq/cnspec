resource "azurerm_servicebus_namespace" "pass" {
  name                = "example-namespace"
  location            = "eastus"
  resource_group_name = "example-rg"
  sku                 = "Premium"

  customer_managed_key {
    key_vault_key_id                  = "https://myvault.vault.azure.net/keys/mykey"
    identity_id                       = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/example"
    infrastructure_encryption_enabled = true
  }
}
