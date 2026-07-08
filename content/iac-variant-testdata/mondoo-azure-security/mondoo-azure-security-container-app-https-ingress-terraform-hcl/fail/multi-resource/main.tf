resource "azurerm_container_app" "compliant" {
  name                         = "compliant-app"
  container_app_environment_id = azurerm_container_app_environment.example.id
  resource_group_name          = "example-rg"
  revision_mode                = "Single"

  ingress {
    external_enabled           = true
    target_port                = 8080
    allow_insecure_connections = false

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    container {
      name   = "app"
      image  = "myregistry.azurecr.io/app@sha256:abc123"
      cpu    = 0.25
      memory = "0.5Gi"
    }
  }
}

resource "azurerm_container_app" "violating" {
  name                         = "violating-app"
  container_app_environment_id = azurerm_container_app_environment.example.id
  resource_group_name          = "example-rg"
  revision_mode                = "Single"

  ingress {
    external_enabled           = true
    target_port                = 8080
    allow_insecure_connections = true

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  template {
    container {
      name   = "app"
      image  = "myregistry.azurecr.io/app@sha256:abc123"
      cpu    = 0.25
      memory = "0.5Gi"
    }
  }
}
