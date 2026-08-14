# The legacy service user type carries the same risk and is covered by the
# same control.
resource "snowflake_legacy_service_user" "reporting" {
  name         = "REPORTING_SERVICE"
  default_role = snowflake_role.reporting.name
  password     = var.reporting_service_password
}
