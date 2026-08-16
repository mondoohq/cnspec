resource "azurerm_monitor_log_profile" "example" {
  name = "default"

  categories = [
    "Action",
    "Delete",
    "Write",
  ]

  locations = [
    "global",
    "eastus",
    "westus",
  ]

  retention_policy {
    enabled = true
    days    = 365
  }
}
