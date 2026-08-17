resource "databricks_mws_private_access_settings" "this" {
  private_access_settings_name = "prod"
  region                       = "us-east-1"
  public_access_enabled        = false
  private_access_level         = "ENDPOINT"
}
