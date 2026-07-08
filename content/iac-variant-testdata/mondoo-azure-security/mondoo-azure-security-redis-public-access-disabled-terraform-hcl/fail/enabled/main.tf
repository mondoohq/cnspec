resource "azurerm_redis_cache" "fail" {
  name                          = "example-cache"
  location                      = "eastus"
  resource_group_name           = "example-rg"
  capacity                      = 1
  family                        = "C"
  sku_name                      = "Standard"
  public_network_access_enabled = true
}
