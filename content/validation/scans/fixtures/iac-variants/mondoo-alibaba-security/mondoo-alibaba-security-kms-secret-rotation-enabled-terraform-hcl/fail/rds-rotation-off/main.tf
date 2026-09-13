# An RDS secret supports automatic rotation. Without it the password never
# changes, so any copy taken from a log, a backup, or a developer machine keeps
# working indefinitely.
resource "alicloud_kms_secret" "db_password" {
  secret_name               = "db-password"
  secret_type               = "Rds"
  secret_data               = jsonencode({ Accounts = [{ AccountName = "app", AccountPassword = var.db_password }] })
  extended_config           = jsonencode({ SecretSubType = "SingleUser", DBInstanceId = "rm-example" })
  version_id                = "v1"
  enable_automatic_rotation = false
}
