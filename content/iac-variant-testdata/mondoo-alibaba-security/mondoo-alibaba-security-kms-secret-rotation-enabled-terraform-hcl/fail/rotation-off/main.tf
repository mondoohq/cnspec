# Without rotation the value never changes, so any copy taken from a log, a
# backup, or a developer machine keeps working indefinitely.
resource "alicloud_kms_secret" "db_password" {
  secret_name               = "db-password"
  secret_data               = var.db_password
  version_id                = "v1"
  enable_automatic_rotation = false
}
