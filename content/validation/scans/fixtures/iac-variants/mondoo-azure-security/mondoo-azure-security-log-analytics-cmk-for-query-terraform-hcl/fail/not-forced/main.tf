resource "azurerm_log_analytics_workspace" "fail" {
  name                = "example-law"
  location            = "eastus"
  resource_group_name = "example-rg"
  sku                 = "PerGB2018"
  retention_in_days   = 30

  cmk_for_query_forced = false
}
