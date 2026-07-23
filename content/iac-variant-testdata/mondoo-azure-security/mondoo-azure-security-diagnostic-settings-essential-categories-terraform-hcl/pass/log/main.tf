# Deprecated `log` block form is still valid HCL and matched by the check.
resource "azurerm_monitor_diagnostic_setting" "activity" {
  name                       = "activity-log-to-law"
  target_resource_id         = "/subscriptions/00000000-0000-0000-0000-000000000000"
  log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/law"

  log {
    category = "Administrative"
    enabled  = true
  }

  log {
    category = "Security"
    enabled  = true
  }

  log {
    category = "Alert"
    enabled  = true
  }

  log {
    category = "Policy"
    enabled  = true
  }
}
