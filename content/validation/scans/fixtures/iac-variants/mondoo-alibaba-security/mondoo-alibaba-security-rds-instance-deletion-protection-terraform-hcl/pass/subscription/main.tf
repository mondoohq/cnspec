# A subscription instance cannot hold release protection, and DeleteDBInstance
# refuses to release it, so it is never one delete request away from loss.
resource "alicloud_db_instance" "prod" {
  engine               = "MySQL"
  engine_version       = "8.0"
  instance_type        = "mysql.n2.medium.1"
  instance_storage     = 100
  instance_name        = "prod-mysql"
  vswitch_id           = alicloud_vswitch.db.id
  instance_charge_type = "Prepaid"
  period               = 12
}
