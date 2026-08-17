resource "databricks_service_principal" "jobs" {
  display_name = "jobs runner"
  active       = true
}
