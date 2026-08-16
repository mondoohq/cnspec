resource "azurerm_redis_cache" "fail" {
  name                = "example-cache"
  location            = "eastus"
  resource_group_name = "example-rg"
  capacity            = 1
  family              = "C"
  sku_name            = "Standard"
  enable_non_ssl_port = true
}
