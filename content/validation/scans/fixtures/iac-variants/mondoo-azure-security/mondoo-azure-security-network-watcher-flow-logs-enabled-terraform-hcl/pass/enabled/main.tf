resource "azurerm_network_watcher_flow_log" "example" {
  network_watcher_name = "example-nw"
  resource_group_name  = "example-rg"
  name                 = "example-flowlog"

  network_security_group_id = azurerm_network_security_group.example.id
  storage_account_id        = azurerm_storage_account.example.id
  enabled                   = true

  retention_policy {
    enabled = true
    days    = 90
  }
}
