resource "databricks_mws_workspaces" "this" {
  account_id     = var.account_id
  workspace_name = "prod"
}
