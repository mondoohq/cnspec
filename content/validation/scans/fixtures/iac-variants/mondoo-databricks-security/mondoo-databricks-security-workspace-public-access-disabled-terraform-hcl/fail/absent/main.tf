resource "databricks_mws_private_access_settings" "this" {
  private_access_settings_name = "prod"
  region                       = "us-east-1"
}
