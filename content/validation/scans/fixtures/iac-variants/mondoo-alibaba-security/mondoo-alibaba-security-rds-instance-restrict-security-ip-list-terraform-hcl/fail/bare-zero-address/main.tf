# A bare 0.0.0.0 entry is treated as open, the same as 0.0.0.0/0, by the live
# check against the instance.
resource "alicloud_db_instance" "prod" {
  engine           = "MySQL"
  engine_version   = "8.0"
  instance_type    = "mysql.n2.medium.1"
  instance_storage = 100
  vswitch_id       = alicloud_vswitch.db.id
  security_ips     = ["10.0.0.0/16", "0.0.0.0"]
}
