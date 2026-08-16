# A production database one API call away from deletion.
resource "alicloud_db_instance" "prod" {
  engine              = "MySQL"
  engine_version      = "8.0"
  instance_type       = "mysql.n2.medium.1"
  instance_storage    = 100
  instance_name       = "prod-mysql"
  vswitch_id          = alicloud_vswitch.db.id
  deletion_protection = false
}
