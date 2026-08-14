resource "azurerm_container_app" "api" {
  name                         = "api"
  container_app_environment_id = azurerm_container_app_environment.prod.id
  resource_group_name          = azurerm_resource_group.prod.name
  revision_mode                = "Single"

  ingress {
    external_enabled = true
    target_port      = 8080

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }

    cors {
      allowed_origins = ["https://app.example.com", "https://admin.example.com"]
    }
  }

  template {
    container {
      name   = "api"
      image  = "example.azurecr.io/api:1.4.2"
      cpu    = 0.5
      memory = "1Gi"
    }
  }
}
