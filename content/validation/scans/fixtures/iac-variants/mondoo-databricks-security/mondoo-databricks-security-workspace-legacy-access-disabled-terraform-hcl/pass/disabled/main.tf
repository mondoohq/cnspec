resource "databricks_disable_legacy_access_setting" "this" {
  disable_legacy_access {
    value = true
  }
}
