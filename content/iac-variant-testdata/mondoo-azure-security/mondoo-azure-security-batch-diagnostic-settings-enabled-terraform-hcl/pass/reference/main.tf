resource "azurerm_batch_account" "example" {
  name                = "examplebatchaccount"
  resource_group_name = "example-rg"
  location            = "eastus"
}

resource "azurerm_log_analytics_workspace" "example" {
  name                = "example-law"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "PerGB2018"
}

resource "azurerm_monitor_diagnostic_setting" "example" {
  name                       = "batch-diagnostics"
  target_resource_id         = azurerm_batch_account.example.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.example.id

  enabled_log {
    category = "ServiceLog"
  }

  metric {
    category = "AllMetrics"
  }
}
