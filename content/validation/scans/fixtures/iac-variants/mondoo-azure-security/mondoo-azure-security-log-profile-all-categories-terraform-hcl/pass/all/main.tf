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
  ]

  servicebus_rule_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.EventHub/namespaces/example-ns/authorizationrules/RootManageSharedAccessKey"

  retention_policy {
    enabled = true
    days    = 365
  }
}
