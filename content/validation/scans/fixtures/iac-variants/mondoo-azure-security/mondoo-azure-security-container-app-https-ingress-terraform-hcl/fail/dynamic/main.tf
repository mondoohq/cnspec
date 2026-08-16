variable "ingress_enabled" {
  type    = list(string)
  default = ["public"]
}

resource "azurerm_container_app" "example" {
  name                         = "example-app"
  container_app_environment_id = azurerm_container_app_environment.example.id
  resource_group_name          = "example-rg"
  revision_mode                = "Single"

  dynamic "ingress" {
    for_each = var.ingress_enabled
    content {
      external_enabled           = true
      target_port                = 8080
      allow_insecure_connections = true

      traffic_weight {
        latest_revision = true
        percentage      = 100
      }
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
