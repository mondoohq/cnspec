resource "azurerm_cosmosdb_account" "example" {
  name                = "example-cosmosdb"
  location            = "eastus"
  resource_group_name = "example-rg"
  offer_type          = "Standard"
  kind                = "GlobalDocumentDB"

  consistency_policy {
    consistency_level = "Session"
  }

  geo_location {
    location          = "eastus"
    failover_priority = 0
  }

  cors_rule {
    allowed_origins    = ["*"]
    allowed_methods    = ["GET", "POST", "PUT", "DELETE"]
    allowed_headers    = ["*"]
    exposed_headers    = ["*"]
    max_age_in_seconds = 3600
  }
}
