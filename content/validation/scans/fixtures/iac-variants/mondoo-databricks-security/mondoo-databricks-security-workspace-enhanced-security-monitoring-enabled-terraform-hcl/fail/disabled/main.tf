resource "databricks_enhanced_security_monitoring_workspace_setting" "this" {
  enhanced_security_monitoring_workspace {
    is_enabled = false
  }
}
