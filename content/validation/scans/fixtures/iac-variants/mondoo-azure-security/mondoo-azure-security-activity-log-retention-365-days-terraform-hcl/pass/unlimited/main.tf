resource "azurerm_monitor_log_profile" "pass" {
  name = "example"

  categories = ["Action"]
  locations  = ["eastus"]

  retention_policy {
    enabled = true
    days    = 0
  }
}
