resource "databricks_restrict_workspace_admins_setting" "this" {
  restrict_workspace_admins {
    status = "ALLOW_ALL"
  }
}
