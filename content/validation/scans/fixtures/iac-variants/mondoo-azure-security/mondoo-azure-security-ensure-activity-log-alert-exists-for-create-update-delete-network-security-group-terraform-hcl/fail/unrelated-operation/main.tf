# Alert exists but watches an unrelated operation, so neither NSG write nor delete is covered.
resource "azurerm_monitor_activity_log_alert" "storage" {
  name                = "storage-delete-alert"
  resource_group_name = "example-rg"
  scopes              = ["/subscriptions/00000000-0000-0000-0000-000000000000"]

  criteria {
    category       = "Administrative"
    operation_name = "Microsoft.Storage/storageAccounts/delete"
  }

  action {
    action_group_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Insights/actionGroups/ops"
  }
}
