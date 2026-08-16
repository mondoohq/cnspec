resource "azurerm_monitor_diagnostic_setting" "activity" {
  name                       = "activity-log-to-law"
  target_resource_id         = "/subscriptions/00000000-0000-0000-0000-000000000000"
  log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/law"

  enabled_log {
    category = "Administrative"
  }
}
