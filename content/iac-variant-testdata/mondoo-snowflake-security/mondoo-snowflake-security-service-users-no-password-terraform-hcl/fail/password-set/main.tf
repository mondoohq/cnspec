# A password on a service user is a shared secret that cannot be MFA-protected
# and lands in the Terraform state.
resource "snowflake_service_user" "etl" {
  name         = "ETL_SERVICE"
  default_role = snowflake_role.etl.name
  password     = var.etl_service_password
}
