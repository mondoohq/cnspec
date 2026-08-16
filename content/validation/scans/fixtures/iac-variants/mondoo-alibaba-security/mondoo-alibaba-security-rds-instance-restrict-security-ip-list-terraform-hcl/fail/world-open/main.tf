# 0.0.0.0/0 in the whitelist makes the database reachable from any address that
# can route to it.
resource "alicloud_db_instance" "prod" {
  engine           = "MySQL"
  engine_version   = "8.0"
  instance_type    = "mysql.n2.medium.1"
  instance_storage = 100
  vswitch_id       = alicloud_vswitch.db.id
  security_ips     = ["10.0.0.0/16", "0.0.0.0/0"]
}
