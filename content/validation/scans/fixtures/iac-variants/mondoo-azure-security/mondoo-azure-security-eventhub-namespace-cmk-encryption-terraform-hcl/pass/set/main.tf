resource "azurerm_eventhub_namespace_customer_managed_key" "example" {
  eventhub_namespace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.EventHub/namespaces/example-ehns"

  key_vault_key_ids = [
    "https://example-kv.vault.azure.net/keys/example-key/00000000000000000000000000000000",
  ]
}
