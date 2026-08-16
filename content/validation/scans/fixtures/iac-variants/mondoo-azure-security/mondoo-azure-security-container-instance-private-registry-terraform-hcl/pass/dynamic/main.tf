variable "containers" {
  type = list(object({
    name  = string
    image = string
  }))
  default = [
    {
      name  = "app"
      image = "prodregistry.azurecr.io/frontend:v2.1.0"
    },
    {
      name  = "sidecar"
      image = "prodregistry.azurecr.io/logging-agent:latest"
    }
  ]
}

resource "azurerm_container_group" "example" {
  name                = "example-container-group"
  location            = "eastus"
  resource_group_name = "example-rg"
  os_type             = "Linux"

  dynamic "container" {
    for_each = var.containers
    content {
      name   = container.value.name
      image  = container.value.image
      cpu    = "0.5"
      memory = "1.5"
    }
  }
}
