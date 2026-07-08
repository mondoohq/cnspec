resource "azurerm_batch_account" "example" {
  name                = "examplebatchaccount"
  resource_group_name = "example-rg"
  location            = "eastus"
}

resource "azurerm_storage_account" "example" {
  name                     = "examplestorageacct"
  resource_group_name      = "example-rg"
  location                 = "eastus"
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

resource "azurerm_log_analytics_workspace" "example" {
  name                = "example-law"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "PerGB2018"
}

resource "azurerm_monitor_diagnostic_setting" "example" {
  name                       = "storage-diagnostics"
  target_resource_id         = azurerm_storage_account.example.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.example.id

  metric {
    category = "AllMetrics"
  }
}
