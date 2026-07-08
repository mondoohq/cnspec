resource "azurerm_cosmosdb_account" "example" {
  name                = "example-cosmosdb"
  location            = "eastus"
  resource_group_name = "example-rg"
  offer_type          = "Standard"
  kind                = "GlobalDocumentDB"

  local_authentication_disabled = false

  consistency_policy {
    consistency_level = "Session"
  }

  geo_location {
    location          = "eastus"
    failover_priority = 0
  }
}
