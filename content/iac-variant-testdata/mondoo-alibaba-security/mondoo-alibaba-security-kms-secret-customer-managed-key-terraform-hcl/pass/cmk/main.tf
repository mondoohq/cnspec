# The secret is protected by a key the account controls and can disable.
resource "alicloud_kms_secret" "db_password" {
  secret_name       = "db-password"
  secret_data       = var.db_password
  version_id        = "v1"
  encryption_key_id = "key-example"
}
