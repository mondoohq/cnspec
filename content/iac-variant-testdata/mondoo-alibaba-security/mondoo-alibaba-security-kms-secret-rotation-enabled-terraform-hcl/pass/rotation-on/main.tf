# The secret rotates every 30 days, so a leaked copy stops working on its own.
resource "alicloud_kms_secret" "db_password" {
  secret_name               = "db-password"
  secret_data               = var.db_password
  version_id                = "v1"
  enable_automatic_rotation = true
  rotation_interval         = "2592000s"
}
