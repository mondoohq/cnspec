# Managed Redis without a customer_managed_key block falls back to a
# platform-managed key, leaving key lifecycle control with Microsoft.
resource "azurerm_managed_redis" "example" {
  name                = "example-managed-redis"
  location            = "eastus"
  resource_group_name = "example-resources"
  sku_name            = "Balanced_B0"
}
