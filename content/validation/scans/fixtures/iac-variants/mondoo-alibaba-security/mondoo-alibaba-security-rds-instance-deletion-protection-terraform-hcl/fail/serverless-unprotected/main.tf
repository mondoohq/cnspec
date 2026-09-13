# Serverless instances support release protection, so they are not exempt.
resource "alicloud_db_instance" "prod" {
  engine               = "MySQL"
  engine_version       = "8.0"
  instance_type        = "mysql.n2.serverless.1c"
  instance_storage     = 30
  instance_name        = "prod-mysql"
  vswitch_id           = alicloud_vswitch.db.id
  instance_charge_type = "Serverless"
  category             = "serverless_basic"
}
