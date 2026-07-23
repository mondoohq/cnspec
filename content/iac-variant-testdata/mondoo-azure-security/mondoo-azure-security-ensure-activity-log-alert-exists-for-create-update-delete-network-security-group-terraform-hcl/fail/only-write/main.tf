# Only the write alert exists; no alert covers NSG delete operations.
resource "azurerm_monitor_activity_log_alert" "nsg_write" {
  name                = "nsg-write-alert"
  resource_group_name = "example-rg"
  scopes              = ["/subscriptions/00000000-0000-0000-0000-000000000000"]

  criteria {
    category       = "Administrative"
    operation_name = "Microsoft.Network/networkSecurityGroups/write"
  }

  action {
    action_group_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Insights/actionGroups/ops"
  }
}
