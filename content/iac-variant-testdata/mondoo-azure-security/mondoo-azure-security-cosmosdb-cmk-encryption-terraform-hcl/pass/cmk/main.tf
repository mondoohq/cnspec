resource "azurerm_cosmosdb_account" "example" {
  name                = "example-cosmosdb"
  location            = "eastus"
  resource_group_name = "example-rg"
  offer_type          = "Standard"
  kind                = "GlobalDocumentDB"

  key_vault_key_id = "https://example-kv.vault.azure.net/keys/cosmos-cmk/abc123def456"

  consistency_policy {
    consistency_level = "Session"
  }

  geo_location {
    location          = "eastus"
    failover_priority = 0
  }
}
