resource "azurerm_container_group" "example" {
  name                = "example-container-group"
  location            = "eastus"
  resource_group_name = "example-rg"
  os_type             = "Linux"

  container {
    name   = "app"
    image  = "nginx:latest"
    cpu    = "0.5"
    memory = "1.5"

    ports {
      port     = 443
      protocol = "TCP"
    }
  }
}
