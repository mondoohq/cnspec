resource "azurerm_container_group" "example" {
  name                = "example-container-group"
  location            = "eastus"
  resource_group_name = "example-rg"
  os_type             = "Linux"

  image_registry_credential {
    server   = "myregistry.azurecr.io"
    username = "myregistry"
    password = var.registry_password
  }

  container {
    name   = "app"
    image  = "myregistry.azurecr.io/app:latest"
    cpu    = "0.5"
    memory = "1.5"

    ports {
      port     = 443
      protocol = "TCP"
    }
  }
}

variable "registry_password" {
  type      = string
  sensitive = true
}
