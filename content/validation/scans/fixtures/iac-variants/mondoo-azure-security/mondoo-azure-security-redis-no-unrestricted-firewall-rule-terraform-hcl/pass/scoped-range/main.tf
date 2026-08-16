resource "azurerm_redis_firewall_rule" "app_tier" {
  name                = "app_tier"
  redis_cache_name    = azurerm_redis_cache.prod.name
  resource_group_name = azurerm_resource_group.prod.name
  start_ip            = "10.0.1.0"
  end_ip              = "10.0.1.255"
}
