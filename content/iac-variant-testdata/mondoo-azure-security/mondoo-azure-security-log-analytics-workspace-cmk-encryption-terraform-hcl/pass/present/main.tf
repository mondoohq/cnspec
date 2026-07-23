resource "azurerm_log_analytics_workspace" "example" {
  name                = "example-law"
  location            = "eastus"
  resource_group_name = "example-rg"
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

resource "azurerm_log_analytics_cluster" "example" {
  name                = "example-cluster"
  resource_group_name = "example-rg"
  location            = "eastus"

  identity {
    type = "SystemAssigned"
  }
}

resource "azurerm_log_analytics_cluster_customer_managed_key" "example" {
  log_analytics_cluster_id = azurerm_log_analytics_cluster.example.id
  key_vault_key_id         = "https://example-kv.vault.azure.net/keys/example-key/abc123"
}
