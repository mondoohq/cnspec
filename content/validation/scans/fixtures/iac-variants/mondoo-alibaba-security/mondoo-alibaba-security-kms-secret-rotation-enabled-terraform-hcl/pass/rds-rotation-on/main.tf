# The RDS secret rotates every 30 days, so a leaked copy of the database
# password stops working on its own.
resource "alicloud_kms_secret" "db_password" {
  secret_name               = "db-password"
  secret_type               = "Rds"
  secret_data               = jsonencode({ Accounts = [{ AccountName = "app", AccountPassword = var.db_password }] })
  extended_config           = jsonencode({ SecretSubType = "SingleUser", DBInstanceId = "rm-example" })
  version_id                = "v1"
  enable_automatic_rotation = true
  rotation_interval         = "2592000s"
}
