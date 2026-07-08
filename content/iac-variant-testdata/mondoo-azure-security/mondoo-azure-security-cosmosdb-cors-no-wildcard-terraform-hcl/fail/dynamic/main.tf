variable "cors_rules" {
  type = list(object({
    allowed_origins = list(string)
  }))
  default = [
    {
      allowed_origins = ["*"]
    }
  ]
}

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

  dynamic "cors_rule" {
    for_each = var.cors_rules
    content {
      allowed_origins    = cors_rule.value.allowed_origins
      allowed_methods    = ["GET", "POST"]
      allowed_headers    = ["*"]
      exposed_headers    = ["*"]
      max_age_in_seconds = 3600
    }
  }
}
