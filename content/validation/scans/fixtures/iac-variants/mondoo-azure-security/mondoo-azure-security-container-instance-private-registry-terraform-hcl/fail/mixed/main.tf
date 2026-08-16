resource "azurerm_container_group" "example" {
  name                = "example-container-group"
  location            = "eastus"
  resource_group_name = "example-rg"
  os_type             = "Linux"

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

  container {
    name   = "cache"
    image  = "redis:7-alpine"
    cpu    = "0.25"
    memory = "0.5"
  }
}
