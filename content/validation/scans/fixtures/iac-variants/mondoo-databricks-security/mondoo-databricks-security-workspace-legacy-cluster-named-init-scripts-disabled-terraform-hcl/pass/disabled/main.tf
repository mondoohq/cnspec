resource "databricks_workspace_conf" "this" {
  custom_config = {
    "enableDeprecatedClusterNamedInitScripts" = "false"
  }
}
