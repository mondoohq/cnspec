# azurerm v4 renamed enable_authentication to authentication_enabled.
resource "azurerm_redis_cache" "fail" {
  name                = "example-cache"
  location            = "eastus"
  resource_group_name = "example-rg"
  capacity            = 1
  family              = "C"
  sku_name            = "Standard"

  redis_configuration {
    authentication_enabled = false
  }
}
