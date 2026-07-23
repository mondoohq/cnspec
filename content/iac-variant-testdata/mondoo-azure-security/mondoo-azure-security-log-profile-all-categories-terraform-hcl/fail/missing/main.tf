resource "azurerm_monitor_log_profile" "example" {
  name = "default"

  categories = [
    "Write",
    "Delete",
  ]

  locations = [
    "global",
    "eastus",
  ]

  retention_policy {
    enabled = true
    days    = 365
  }
}
