# The CMK association is declared but names no key, so the cluster keeps
# Microsoft-managed encryption.
resource "azurerm_kusto_cluster_customer_managed_key" "analytics" {
  cluster_id   = azurerm_kusto_cluster.analytics.id
  key_vault_id = azurerm_key_vault.prod.id
  key_name     = ""
}
