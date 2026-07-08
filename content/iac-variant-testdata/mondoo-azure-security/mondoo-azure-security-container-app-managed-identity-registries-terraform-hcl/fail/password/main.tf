resource "azurerm_container_app" "example" {
  name                         = "example-app"
  container_app_environment_id = azurerm_container_app_environment.example.id
  resource_group_name          = "example-rg"
  revision_mode                = "Single"

  secret {
    name  = "registry-password"
    value = "s3cr3t"
  }

  registry {
    server               = "myregistry.azurecr.io"
    username             = "admin"
    password_secret_name = "registry-password"
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
