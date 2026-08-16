# 0.0.0.0 through 255.255.255.255 is the whole IPv4 space, which is the same as
# having no firewall at all.
resource "azurerm_redis_firewall_rule" "allow_all" {
  name                = "allow_all"
  redis_cache_name    = azurerm_redis_cache.prod.name
  resource_group_name = azurerm_resource_group.prod.name
  start_ip            = "0.0.0.0"
  end_ip              = "255.255.255.255"
}
