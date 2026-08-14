resource "snowflake_service_user" "etl" {
  name         = "ETL_SERVICE"
  default_role = snowflake_role.etl.name
  default_warehouse = snowflake_warehouse.etl.name

  # Key-pair authentication, no password on the account.
  rsa_public_key = file("${path.module}/keys/etl_rsa.pub")
  comment        = "Runs the nightly ETL jobs"
}
