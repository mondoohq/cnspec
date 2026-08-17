resource "databricks_obo_token" "jobs" {
  application_id = databricks_service_principal.jobs.application_id
  comment        = "jobs runner"
}
