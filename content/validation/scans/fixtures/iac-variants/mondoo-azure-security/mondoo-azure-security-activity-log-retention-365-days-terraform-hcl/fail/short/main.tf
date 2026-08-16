resource "azurerm_monitor_log_profile" "fail" {
  name = "example"

  categories = ["Action"]
  locations  = ["eastus"]

  retention_policy {
    enabled = true
    days    = 90
  }
}
