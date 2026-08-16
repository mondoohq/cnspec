variable "containers" {
  type = list(object({
    name  = string
    image = string
  }))
  default = [
    {
      name  = "app"
      image = "myregistry.azurecr.io/app@sha256:5b8d5f2c4e1a9b0c3d7e6f8a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c"
    }
  ]
}

resource "azurerm_container_app" "example" {
  name                         = "example-app"
  container_app_environment_id = azurerm_container_app_environment.example.id
  resource_group_name          = "example-rg"
  revision_mode                = "Single"

  template {
    dynamic "container" {
      for_each = var.containers
      content {
        name   = container.value.name
        image  = container.value.image
        cpu    = 0.25
        memory = "0.5Gi"
      }
    }
  }
}
