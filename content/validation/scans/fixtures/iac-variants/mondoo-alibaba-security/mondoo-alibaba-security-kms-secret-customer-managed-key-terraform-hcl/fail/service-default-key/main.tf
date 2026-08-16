# With no key named, the secret falls back to the service default, so the account
# has no key policy, no separate audit trail, and no way to revoke by disabling.
resource "alicloud_kms_secret" "db_password" {
  secret_name = "db-password"
  secret_data = var.db_password
  version_id  = "v1"
}
