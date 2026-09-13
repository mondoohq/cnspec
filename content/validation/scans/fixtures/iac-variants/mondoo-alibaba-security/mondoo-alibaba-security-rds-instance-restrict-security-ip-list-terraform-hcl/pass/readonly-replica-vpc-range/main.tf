resource "alicloud_db_instance" "prod" {
  engine           = "MySQL"
  engine_version   = "8.0"
  instance_type    = "mysql.n2.medium.1"
  instance_storage = 100
  vswitch_id       = alicloud_vswitch.db.id
  security_ips     = ["10.0.0.0/16"]
}

resource "alicloud_db_readonly_instance" "replica" {
  master_db_instance_id = alicloud_db_instance.prod.id
  engine_version        = "8.0"
  instance_type         = "mysql.n2.medium.1"
  instance_storage      = 100
  vswitch_id            = alicloud_vswitch.db.id
  security_ips          = ["10.0.0.0/16", "10.1.0.0/16"]
}
