resource "azurerm_kusto_cluster_customer_managed_key" "analytics" {
  cluster_id   = azurerm_kusto_cluster.analytics.id
  key_vault_id = azurerm_key_vault.prod.id
  key_name     = azurerm_key_vault_key.kusto.name
  key_version  = azurerm_key_vault_key.kusto.version
}
