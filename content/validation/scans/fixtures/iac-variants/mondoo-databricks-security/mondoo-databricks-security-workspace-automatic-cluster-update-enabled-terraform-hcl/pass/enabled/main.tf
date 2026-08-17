resource "databricks_automatic_cluster_update_workspace_setting" "this" {
  automatic_cluster_update_workspace {
    enabled = true
  }
}
